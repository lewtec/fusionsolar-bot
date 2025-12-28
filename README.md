# fusionsolar-bot

Puxa os dados de produção do dia do Fusion Solar e manda para a lista de e-mails indicada.

## Como "instalar"

### Docker
Para rodar a versão mais recente diretamente do registro:

```bash
docker run ghcr.io/lucasew/fusionsolar-bot:latest [parametros]
```

Para construir a imagem localmente:

```bash
docker build -t fusionsolar-bot .
```

E para rodar a imagem construída localmente:

```bash
docker run fusionsolar-bot [parametros]
```

Nenhum estado desse container precisa ser salvo.

### Conda/pip
Você vai precisar do `chromium/chrome`, `chromedriver` e do `selenium`. O sistema de envio de e-mail já tá no Python.

### GitHub Actions
Considerado experimental.

Por algum motivo o sistema não consegue acessar o fusionsolar pelo GitHub Actions. Cheguei a tentar pelo Tor sem muito sucesso.

O workflow, na fase de configurar o sistema que oculta os secrets, acaba mostrando os secrets :facepalm:

## Parâmetros
Esse projeto faz basicamente duas coisas:
- Pega os dados de produção de todas as estações em uma conta fusionsolar
- Envia os dados por e-mail para uma lista de emails definida

Se as informações sobre o e-mail estão incompetas ele busca as informações igual, só não manda email. Bom pra testar.

Parâmetros no formato `flag/variável de ambiente`.

- `--user/FUSIONSOLAR_USER`: usuário para logar no fusionsolar, usuários errados podem falhar silenciosamente
- `--password/FUSIONSOLAR_PASSWORD`: usuário para logar no fusionsolar, senhas erradas podem falhar silenciosamente
- `--smtp-user/SMTP_USER`: usuário para logar no SMTP do servidor para enviar e-mail
- `--smtp-password/SMTP_PASSWD`: senha para logar no SMTP do servidor para enviar e-mail
- `--smtp-server/SMTP_SERVER`: servidor SMTP para envio do email
- `--smtp-destinations/SMTP_DESTINATIONS`: lista de e-mails para enviar os resultados separada por espaço
- `--headless`: não mostar janela do chrome usada na automação, usada internamente
- `--verbose`: dar mais detalhes sobre o que tá acontecendo, bom pra debug

## Recomendações
- Não use G-Mail como provedor de SMTP, muito menos sua conta pessoal. Eu uso uma conta
gmx.com para isso. Todo dia vai pelo menos um email para mim e para o meu pai e nunca
caiu no spam, e não preciso comprometer minha conta pessoal.
- Se o envio/extração falhar, provavelmente é por causa de um modal corno e de tempos em tempos
isso acontece, por exemplo, quando eles trocam os termos.

## O que falta ser feito?
- [ ] Endpoints alternativos: por enquanto a aplicação tá chumbada pra usar o endpoint internacional.
- [ ] Deixar escalável 😂😂😂
