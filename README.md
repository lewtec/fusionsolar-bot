# fusionsolar-bot

Puxa os dados de produção do dia do Fusion Solar e manda para a lista de e-mails indicada.

Repositório: [lewtec/fusionsolar-bot](https://github.com/lewtec/fusionsolar-bot).
O navegador **não** vai na imagem — use um endpoint CDP remoto (`BROWSER_CDP`).

## Como "instalar"

### Docker

Versão mais recente no GHCR:

```bash
docker run --rm \
  -e BROWSER_CDP="ws://your-cdp-endpoint:9222/devtools/browser/abc123" \
  -e FUSIONSOLAR_USER=... \
  -e FUSIONSOLAR_PASSWORD=... \
  ghcr.io/lewtec/fusionsolar-bot:latest
```

Imagens e binários Linux (amd64/arm64) saem no [GitHub Releases](https://github.com/lewtec/fusionsolar-bot/releases) via GoReleaser.
A imagem de runtime só copia o binário estático (Alpine, non-root); o build é do GoReleaser, não de um `docker build` multi-stage local.

### Go (binário / source)

Este projeto usa [mise](https://mise.jdx.dev) para Go e ferramentas de release.

```bash
mise install
go run ./cmd/fusionsolar-bot [parametros]
# ou
go install github.com/lewtec/fusionsolar-bot/cmd/fusionsolar-bot@latest
```

Configure `BROWSER_CDP` (ou `--browser-cdp`) antes de rodar.

### Release (maintainers)

Versionamento = tags git via [svu](https://github.com/caarlos0/svu) (sem prefixo `v`). Ver [SPEC.md](SPEC.md).

```bash
mise release next    # ou patch | minor | major
```

No GitHub: **Actions → Autorelease → Run workflow** com o bump desejado. Push/PR só rodam CI (`mise run ci`).

### GitHub Actions

Experimental. A action composta puxa `ghcr.io/lewtec/fusionsolar-bot` com a **mesma ref** do `uses:` (tag `0.7.0` → imagem `:0.7.0`; branch `main` → `:latest`). Ela **não** sobe o browser — passe `browser_cdp`.

`browser_cdp` tem de ser alcançável **de dentro do container do bot** (a action faz `docker run`). O hostname de um `services:` do job (ex.: `browserless`) **não** resolve na bridge padrão do Docker — use um endpoint CDP público/privado que o container consiga rotear, ou rode o browser no host e aponte para um IP/porta acessível de lá.

Inputs opcionais (`smtp_from`, `sentry_dsn`, `timeout`, `max_login_retries`, `verbose`) existem no `action.yml` atual de `main` (e em tags posteriores a `0.7.0`). A tag `0.6.1` ainda apontava para a imagem antiga e não expõe esses inputs.

```yaml
name: fusionsolar-report
on:
  workflow_dispatch:

jobs:
  report:
    runs-on: ubuntu-latest
    steps:
      - name: Run fusionsolar report
        # Prefer a released tag once it includes the optional inputs below;
        # @main tracks the full Action surface and :latest image.
        uses: lewtec/fusionsolar-bot@main
        with:
          user: ${{ secrets.FUSIONSOLAR_USER }}
          password: ${{ secrets.FUSIONSOLAR_PASSWORD }}
          # Must be reachable from inside the bot container (not only the job VM).
          browser_cdp: ${{ secrets.BROWSER_CDP }}
          smtp_user: ${{ secrets.SMTP_USER }}
          smtp_from: ${{ secrets.SMTP_FROM }}
          smtp_passwd: ${{ secrets.SMTP_PASSWD }}
          smtp_server: ${{ secrets.SMTP_SERVER }}
          smtp_destinations: ${{ secrets.SMTP_DESTINATIONS }}
          # opcionais: sentry_dsn, timeout, max_login_retries, verbose
```

Inputs opcionais espelham as mesmas variáveis de ambiente do binário
(`SMTP_FROM`, `SENTRY_DSN`, `TIMEOUT`, `MAX_LOGIN_RETRIES`, `VERBOSE`).

## Parâmetros

Esse projeto faz basicamente duas coisas:
- Pega os dados de produção de todas as estações em uma conta fusionsolar
- Envia os dados por e-mail para uma lista de emails definida

Se as informações sobre o e-mail estão incompletas ele busca as informações igual, só não manda email. Bom pra testar.

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
- `--max-login-retries` / `MAX_LOGIN_RETRIES`: tentativas de login antes de falhar, padrão: 5
- `--verbose` / `VERBOSE`: dar mais detalhes sobre o que está acontecendo, bom para debug
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
