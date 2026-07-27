package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	allowedAction    = "KILL_ANYDESK"
	maxMessageBytes  = 8 * 1024
	reconnectMaxWait = time.Minute
)

type config struct {
	MachineID string `json:"machineId"`
	SSH       struct {
		Host       string `json:"host"`
		Port       int    `json:"port"`
		Username   string `json:"username"`
		ClientKey  string `json:"clientKey"`
		KnownHosts string `json:"knownHosts"`
	} `json:"ssh"`
	Redis struct {
		RemoteHost string `json:"remoteHost"`
		RemotePort int    `json:"remotePort"`
	} `json:"redis"`
	LogFile string `json:"logFile"`
}

type command struct {
	Version       int             `json:"version"`
	CommandID     string          `json:"commandId"`
	CorrelationID string          `json:"correlationId"`
	Target        string          `json:"target"`
	Action        string          `json:"action"`
	Args          json.RawMessage `json:"args"`
	RequestedAt   time.Time       `json:"requestedAt"`
	ExpiresAt     time.Time       `json:"expiresAt"`
}

type outcome struct {
	Matched     int `json:"matched"`
	ForceKilled int `json:"forceKilled"`
}

type commandRunner func(string, ...string) (string, error)

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil { return config{}, err }
	var c config
	if err := json.Unmarshal(data, &c); err != nil { return config{}, err }
	if c.MachineID == "" || c.SSH.Host == "" || c.SSH.Username == "" { return config{}, errors.New("machineId and SSH host/username are required") }
	if c.SSH.Port == 0 { c.SSH.Port = 22 }
	if c.Redis.RemoteHost == "" { c.Redis.RemoteHost = "127.0.0.1" }
	if c.Redis.RemotePort == 0 { c.Redis.RemotePort = 6379 }
	if c.LogFile == "" { c.LogFile = "logs/anydesk-agent.log" }
	if _, err := os.Stat(c.SSH.ClientKey); err != nil { return config{}, fmt.Errorf("SSH clientKey: %w", err) }
	if _, err := os.Stat(c.SSH.KnownHosts); err != nil { return config{}, fmt.Errorf("SSH knownHosts: %w", err) }
	return c, nil
}

func validateCommand(raw, machineID string, now time.Time) (command, error) {
	if len([]byte(raw)) >= maxMessageBytes { return command{}, errors.New("message is too large") }
	var c command
	if err := json.Unmarshal([]byte(raw), &c); err != nil { return command{}, fmt.Errorf("invalid JSON: %w", err) }
	if c.Version != 1 { return command{}, errors.New("unsupported version") }
	if c.CommandID == "" || c.CorrelationID == "" { return command{}, errors.New("commandId and correlationId are required") }
	if c.Target != machineID { return command{}, errors.New("message target does not match this machine") }
	if c.Action != allowedAction { return command{}, errors.New("action is not allowed") }
	if string(c.Args) != "{}" { return command{}, errors.New("arguments are not allowed for this action") }
	if c.RequestedAt.Location() != time.UTC || c.ExpiresAt.Location() != time.UTC { return command{}, errors.New("timestamps must be UTC") }
	if !c.ExpiresAt.After(c.RequestedAt) || !c.ExpiresAt.After(now) { return command{}, errors.New("message has expired") }
	return c, nil
}

func killAnyDesk(run commandRunner, tasklistOutput string) (outcome, error) {
	matched := 0
	for _, line := range strings.Split(tasklistOutput, "\n") {
		if strings.Contains(strings.ToLower(line), "anydesk.exe") { matched++ }
	}
	if matched == 0 { return outcome{}, nil }
	_, err := run("taskkill", "/IM", "AnyDesk.exe", "/F")
	if err != nil { return outcome{}, fmt.Errorf("taskkill AnyDesk.exe: %w", err) }
	return outcome{Matched: matched, ForceKilled: matched}, nil
}

func executeKill() (outcome, error) {
	list := exec.Command("tasklist", "/FI", "IMAGENAME eq AnyDesk.exe", "/FO", "CSV", "/NH")
	output, err := list.Output()
	if err != nil { return outcome{}, fmt.Errorf("list AnyDesk.exe: %w", err) }
	return killAnyDesk(func(name string, args ...string) (string, error) {
		output, err := exec.Command(name, args...).CombinedOutput()
		return string(output), err
	}, string(output))
}

func tunnel(client *ssh.Client, remote string) (string, func(), error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { return "", nil, err }
	var once sync.Once
	closeAll := func() { once.Do(func() { _ = listener.Close() }) }
	go func() {
		for {
			local, err := listener.Accept()
			if err != nil { return }
			go func() {
				defer local.Close()
				remoteConn, err := client.Dial("tcp", remote)
				if err != nil { return }
				defer remoteConn.Close()
				go io.Copy(remoteConn, local)
				_, _ = io.Copy(local, remoteConn)
			}()
		}
	}()
	return listener.Addr().String(), closeAll, nil
}

func sshClient(c config) (*ssh.Client, error) {
	key, err := os.ReadFile(c.SSH.ClientKey)
	if err != nil { return nil, err }
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil { return nil, err }
	hostKeyCallback, err := knownhosts.New(c.SSH.KnownHosts)
	if err != nil { return nil, err }
	return ssh.Dial("tcp", net.JoinHostPort(c.SSH.Host, fmt.Sprint(c.SSH.Port)), &ssh.ClientConfig{
		User: c.SSH.Username, Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)}, HostKeyCallback: hostKeyCallback,
		Timeout: 20 * time.Second,
	})
}

func queue(machineID, suffix string) string { return "remote-agent:" + suffix + ":" + machineID }

func consume(ctx context.Context, c config, logger *log.Logger) error {
	client, err := sshClient(c)
	if err != nil { return err }
	defer client.Close()
	address, closeTunnel, err := tunnel(client, net.JoinHostPort(c.Redis.RemoteHost, fmt.Sprint(c.Redis.RemotePort)))
	if err != nil { return err }
	defer closeTunnel()
	redisClient := redis.NewClient(&redis.Options{Addr: address})
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil { return err }
	logger.Printf("connected; waiting on %s", queue(c.MachineID, "commands"))
	for ctx.Err() == nil {
		item, err := redisClient.BRPop(ctx, 30*time.Second, queue(c.MachineID, "commands")).Result()
		if errors.Is(err, redis.Nil) { continue }
		if err != nil { return err }
		raw := item[1]
		cmd, err := validateCommand(raw, c.MachineID, time.Now().UTC())
		if err != nil {
			payload, _ := json.Marshal(map[string]any{"target": c.MachineID, "status": "REJECTED", "reason": err.Error(), "receivedAt": time.Now().UTC(), "raw": raw})
			if publishErr := redisClient.LPush(ctx, queue(c.MachineID, "dead-letter"), payload).Err(); publishErr != nil { return publishErr }
			logger.Printf("command rejected: %v", err); continue
		}
		result, err := executeKill()
		if err != nil { return err }
		payload, _ := json.Marshal(map[string]any{"commandId": cmd.CommandID, "correlationId": cmd.CorrelationID, "target": c.MachineID, "status": "SUCCEEDED", "completedAt": time.Now().UTC(), "outcome": result})
		if err := redisClient.LPush(ctx, queue(c.MachineID, "results"), payload).Err(); err != nil { return err }
		logger.Printf("KILL_ANYDESK completed: matched=%d forceKilled=%d", result.Matched, result.ForceKilled)
	}
	return ctx.Err()
}

func main() {
	configPath := flag.String("config", "config.json", "path to agent configuration")
	flag.Parse()
	c, err := loadConfig(*configPath)
	if err != nil { log.Fatal(err) }
	if err := os.MkdirAll(filepath.Dir(c.LogFile), 0750); err != nil { log.Fatal(err) }
	file, err := os.OpenFile(c.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil { log.Fatal(err) }
	defer file.Close()
	logger := log.New(io.MultiWriter(os.Stdout, file), "anydesk-agent ", log.LstdFlags|log.LUTC)
	for wait := time.Second; ; wait = min(wait*2, reconnectMaxWait) {
		err := consume(context.Background(), c, logger)
		logger.Printf("transport failure; reconnecting in %s: %v", wait, err)
		time.Sleep(wait)
	}
}
