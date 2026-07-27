# Spec — Catálogo de comandos remotos controlados

- **Status:** planejada
- **Origem:** evolução explicitamente fora do escopo do MVP `KILL_ANYDESK`
- **Pré-requisito:** agente Go V1 estável em produção

## Objetivo

Evoluir o agente de uma única ação allowlisted para um catálogo pequeno, explícito e auditável de comandos remotos. A evolução não pode introduzir shell remoto nem execução de texto recebido pela fila.

## Escopo

1. Definir um registry compilado de ações e schemas de argumentos por versão.
2. Adicionar comandos somente após cada um ter uma spec e testes próprios.
3. Migrar transporte para Redis Streams com consumer groups, confirmação (`XACK`) e recuperação de pendências antes de comandos não idempotentes.
4. Incluir resultado estruturado, correlação, expiração, deduplicação por `commandId` e retenção limitada.
5. Expor no n8n apenas opções do catálogo e campos validados — nunca um campo livre de comando.
6. Aplicar ACL Redis por agente e, se necessário, assinatura criptográfica de envelopes.
7. Criar auditoria de quem solicitou, alvo, ação, resultado e timestamps, sem segredos nem payloads sensíveis.

## Não escopo

- PowerShell, CMD, Bash ou qualquer shell genérico;
- terminal interativo;
- upload/download de arquivos;
- captura ou controle de tela, teclado, mouse, áudio ou webcam;
- elevação de privilégio, bypass de UAC ou instalação de serviço;
- autoatualização nesta primeira evolução.

## Regras de segurança obrigatórias

- O agente recebe somente JSON e despacha apenas handlers compilados.
- Cada action declara schema, autorização, expiração máxima e semântica de idempotência.
- Não usar `exec`, `eval`, `os/exec` com strings recebidas nem shell.
- O workflow n8n não decide comandos arbitrários: ele apenas cria envelopes permitidos.
- Toda action nova precisa de threat model curto e testes de rejeição de payload inválido.

## Critérios de aceite

- Uma action desconhecida ou com argumentos fora do schema é rejeitada e auditada.
- Mensagens duplicadas não repetem uma ação não idempotente.
- Mensagens pendentes são recuperáveis após reinício do agente.
- O n8n mostra apenas ações autorizadas para cada alvo.
- Não existe caminho de execução genérica de processos ou shell.

## Decisão pendente

Definir o primeiro comando adicional, a justificativa operacional e se ele é idempotente. Sem isso, nenhuma ação além de `KILL_ANYDESK` será implementada.
