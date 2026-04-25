# fusionsolar-bot

Puxa os dados de produção do dia do Fusion Solar e manda para a lista de e-mails indicada.

## Como "instalar"

### Docker
Para rodar a versão mais recente diretamente do registro:

```bash
docker run -e BROWSER_CDP="ws://your-cdp-endpoint:9222/devtools/browser/abc123" ghcr.io/lucasew/fusionsolar-bot:latest [parametros]
```

Para construir a imagem localmente:

```bash
docker build -t fusionsolar-bot .
```

E para rodar a imagem construída localmente:

```bash
docker run -e BROWSER_CDP="ws://your-cdp-endpoint:9222/devtools/browser/abc123" fusionsolar-bot [parametros]
```

Nenhum estado desse container precisa ser salvo.

### Go (Binário/Source)
Este projeto usa o [mise](https://mise.jdx.dev) para gerenciar a versão do Go. Se você tiver o `mise` instalado, basta entrar na pasta do projeto e ele baixará a versão correta do Go automaticamente.

Este projeto depende de um navegador exposto via CDP. Configure a variável `BROWSER_CDP` com a URL do endpoint antes de rodar.

Para rodar diretamente do código fonte:
```bash
go run ./cmd/fusionsolar-bot [parametros]
```

Para compilar e instalar:
```bash
go install ./cmd/fusionsolar-bot
fusionsolar-bot [parametros]
```

### GitHub Actions
Considerado experimental.

Por algum motivo o sistema não consegue acessar o fusionsolar pelo GitHub Actions. Cheguei a tentar pelo Tor sem muito sucesso.

O workflow, na fase de configurar o sistema que oculta os secrets, acaba mostrando os secrets :facepalm:

## Parâmetros
Esse projeto faz basicamente duas coisas:
- Pega os dados de produção de todas as estações em uma conta fusionsolar
- Envia os dados por e-mail para uma lista de emails definida

Se as informações sobre o e-mail estão incompetas ele busca as informações igual, só não manda email. Bom pra testar.

Parâmetros no formato `flag` / `variável de ambiente`.

- `--user` / `FUSIONSOLAR_USER`: usuário para logar no fusionsolar
- `--password` / `FUSIONSOLAR_PASSWORD`: senha para logar no fusionsolar
- `--smtp-user` / `SMTP_USER`: usuário para logar no SMTP do servidor para enviar e-mail
- `--smtp-from` / `SMTP_FROM`: remetente do e-mail (opcional, padrão: smtp-user)
- `--smtp-passwd` / `SMTP_PASSWD`: senha para logar no SMTP do servidor para enviar e-mail
- `--smtp-server` / `SMTP_SERVER`: servidor SMTP para envio do email (host:port)
- `--smtp-destinations` / `SMTP_DESTINATIONS`: lista de e-mails para enviar os resultados separada por espaço
- `--sentry-dsn` / `SENTRY_DSN`: DSN do Sentry para monitoramento de erros
- `--timeout` / `TIMEOUT`: tempo máximo total de execução antes de cancelar o trabalho, padrão: 10 minutos
- `--browser-cdp` / `BROWSER_CDP`: endpoint CDP do navegador remoto (obrigatório)
- `--verbose` (apenas flag): dar mais detalhes sobre o que está acontecendo, bom para debug
- `--version`: exibe a versão do programa

## Recomendações
- Não use G-Mail como provedor de SMTP, muito menos sua conta pessoal. Eu uso uma conta
gmx.com para isso. Todo dia vai pelo menos um email para mim e para o meu pai e nunca
caiu no spam, e não preciso comprometer minha conta pessoal.
- Se o envio/extração falhar, provavelmente é por causa de um modal corno e de tempos em tempos
isso acontece, por exemplo, quando eles trocam os termos.

## O que falta ser feito?
- [ ] Endpoints alternativos: por enquanto a aplicação tá chumbada pra usar o endpoint internacional.
- [ ] Deixar escalável 😂😂😂
