import subprocess
import tempfile
import os
import time
import base64
import smtplib
import ssl
from argparse import ArgumentParser
from datetime import datetime
from shutil import which
from sys import stderr

from email import encoders
from email.mime.base import MIMEBase
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText

from selenium import webdriver
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.common.by import By
from selenium.webdriver.chrome.service import Service

def setup_sentry(dsn):
    """Configure Sentry if available and DSN is provided."""
    try:
        import sentry_sdk
        print("[*] Configurando sentry")
        if dsn is not None:
            sentry_sdk.init(
                dsn=dsn,
                traces_sample_rate=1.0,
                _experiments={"continuous_profiling_auto_start": True}
            )
        else:
            print("[!] Sentry: DSN não especificado")
    except ImportError:
        print("[*] Configuração do Sentry ignorada: SDK não instalado")

def setup_chrome_driver(tmpdir, proxy, headless, verbose):
    """Configure and return a Chrome WebDriver instance."""
    service_args = ['--verbose'] if verbose else []
    service = Service(
        executable_path=which("chromedriver"),
        service_args=service_args,
        log_output=subprocess.STDOUT
    )

    options = webdriver.ChromeOptions()
    if proxy is not None:
        options.add_argument(f'--proxy-server={proxy}')

    if headless:
        options.add_argument('--headless')
        options.add_argument('--disable-gpu')
        options.add_argument('--disable-dev-shm-usage')
        options.add_argument('--no-sandbox')

    if verbose:
        options.add_argument('--log-level=DEBUG')

    options.add_argument(f'--user-data-dir={tmpdir}')
    options.add_argument('--window-size=1280,720')
    options.add_argument('--remote-debugging-pipe')

    return webdriver.Chrome(options=options, service=service)

def login_to_fusionsolar(driver, username, password):
    """Login to FusionSolar website."""
    while True:
        print('[*] Login', file=stderr)
        driver.get("https://intl.fusionsolar.huawei.com/pvmswebsite/login/build/index.html#/LOGIN")
        time.sleep(10)

        driver.find_element(By.CSS_SELECTOR, "div#username input").send_keys(username)
        password_input = driver.find_element(By.CSS_SELECTOR, "div#password input")
        password_input.send_keys(password)
        password_input.send_keys(Keys.ENTER)
        time.sleep(10)

        # Handle cookie dialog
        cookie_buttons = driver.find_elements(By.CSS_SELECTOR, "i.cookiePolicy-icon")
        for button in cookie_buttons:
            print('[*] Fechando diálogo de cookie', file=stderr)
            button.click()

        # Handle privacy modal
        for modal in driver.find_elements(By.CSS_SELECTOR, "div.nco-privacy-confirm-modal"):
            if modal.find_element(By.CSS_SELECTOR, 'div.nco-privacy-content') is None:
                continue
            approve_button = modal.find_element(By.CSS_SELECTOR, 'button.dpdesign-btn-primary')
            if approve_button:
                time.sleep(1)
                approve_button.click()
                time.sleep(10)

        if 'login' not in driver.current_url:
            break
        else:
            print('[*] Reiniciando processo de login', file=stderr)

def get_stations(driver, max_attempts=5):
    """Get all stations from the dashboard."""
    stations_data = []
    attempts = 0

    while len(stations_data) == 0 and attempts < max_attempts:
        print('[*] Acessando homepage', file=stderr)
        driver.get("https://intl.fusionsolar.huawei.com")
        time.sleep(10)

        print('[*] Tentando listar estações', file=stderr)
        stations = driver.find_elements(By.CSS_SELECTOR, "a.nco-home-list-text-ellipsis")

        if not stations:
            attempts += 1
            print(f"[*] Tentativa {attempts}/{max_attempts}: Zero estações encontradas", file=stderr)
            print(f"[*] URL atual: {driver.current_url}", file=stderr)
            if attempts >= max_attempts:
                print('[*] Sem estações, desistindo...', file=stderr)
                exit(1)
        else:
            for station in stations:
                station_data = {
                    "url": station.get_attribute('href'),
                    "name": station.text
                }
                print(station_data, file=stderr)
                stations_data.append(station_data)

    return stations_data

def collect_station_data(driver, stations_data):
    """Collect energy production data from each station."""
    email_text = ["Quantidade de energia produzida em cada base", ""]
    attachments = []

    for station in stations_data:
        station_url = station['url']
        station_name = station['name']

        print(f'[*] Coletando dados da estação "{station_name}"', file=stderr)
        driver.get(station_url)
        time.sleep(10)

        # Get canvas chart
        canvas = driver.find_element(By.CSS_SELECTOR, ".nco-single-energy-body canvas")
        canvas_b64 = driver.execute_script("return arguments[0].toDataURL('image/png').substring(21);", canvas)

        # Get production amount
        amount_produced = float(driver.find_element(By.CSS_SELECTOR, "span.value").text.replace(',', '.'))
        email_text.append(f"{station_name}: {amount_produced}kWh")
        print(f'[*] Produzido hoje: {amount_produced}kWh', file=stderr)

        # Create image attachment
        image = MIMEBase("image", "png")
        image.set_payload(base64.b64decode(canvas_b64))
        encoders.encode_base64(image)
        image.add_header("Content-Disposition", f"attachment; filename= {station_name}.png")
        attachments.append(image)

    email_text.extend([
        "",
        "Os gráficos de geração estão em anexo.",
        "",
        f"Dados obtidos em: {datetime.today()}"
    ])

    return email_text, attachments

def send_email(smtp_user, smtp_from, smtp_passwd, smtp_server, destinations, subject, body_text, attachments):
    """Send email with attachments."""
    message = MIMEMultipart()
    message['From'] = smtp_from
    message['To'] = destinations.replace(' ', ', ')
    message['Subject'] = subject

    message.attach(MIMEText(body_text, 'plain'))
    for attachment in attachments:
        message.attach(attachment)

    context = ssl.create_default_context()

    server_parts = smtp_server.split(":")
    server_name = server_parts[0]
    server_port = int(server_parts[1]) if len(server_parts) > 1 else 587

    print('[*] Enviando emails', file=stderr)
    with smtplib.SMTP(server_name, server_port) as server:
        server.starttls(context=context)
        server.login(smtp_user, smtp_passwd)
        server.sendmail(smtp_user, destinations.split(), message.as_string())

def main():
    parser = ArgumentParser(description="FusionSolar data collector")
    parser.add_argument("--user", default=os.getenv("FUSIONSOLAR_USER"), help="FusionSolar username")
    parser.add_argument("--password", default=os.getenv("FUSIONSOLAR_PASSWORD"), help="FusionSolar password")
    parser.add_argument('--smtp-user', default=os.getenv("SMTP_USER"), help="SMTP username")
    parser.add_argument('--smtp-from', default=os.environ.get("SMTP_FROM", os.getenv("SMTP_USER")), help="SMTP username")
    parser.add_argument('--smtp-passwd', default=os.getenv("SMTP_PASSWD"), help="SMTP password")
    parser.add_argument('--smtp-server', default=os.getenv("SMTP_SERVER"), help="SMTP server (format: server:port)")
    parser.add_argument('--smtp-destinations', default=os.getenv("SMTP_DESTINATIONS"), help="Email recipients (space separated)")
    parser.add_argument('--sentry-dsn', default=os.getenv("SENTRY_DSN"), help="Sentry DSN for error tracking")
    parser.add_argument('--proxy', default=os.environ.get("SELENIUM_PROXY_SERVER"), help="Proxy server for Selenium")
    parser.add_argument('--headless', action='store_true', help="Run Chrome in headless mode")
    parser.add_argument('--verbose', '-v', action='store_true', help="Enable verbose logging")
    args = parser.parse_args()

    # Setup Sentry for error tracking
    setup_sentry(args.sentry_dsn)

    # Check if email is enabled
    email_enabled = None not in [args.smtp_user, args.smtp_passwd, args.smtp_server, args.smtp_destinations]
    if not email_enabled:
        print("[!] Funcionalidade de email desativada", file=stderr)

    now = datetime.today()

    # Validate required credentials
    if None in [args.user, args.password]:
        print("[!] Usuário e senha fusionsolar não fornecidos")
        exit(1)

    with tempfile.TemporaryDirectory() as tmpdir:
        # Setup and configure Chrome WebDriver
        driver = setup_chrome_driver(tmpdir, args.proxy, args.headless, args.verbose)

        try:
            # Login to FusionSolar
            login_to_fusionsolar(driver, args.user, args.password)

            # Get stations data
            stations_data = get_stations(driver)

            # Collect data from each station
            email_text, attachments = collect_station_data(driver, stations_data)

            # Print collected data
            print("\n".join(email_text))

            # Send email if enabled
            if email_enabled:
                subject = f"Relatório do dia {str(now).split(' ')[0]} FusionSolar"
                send_email(
                    args.smtp_user,
                    args.smtp_from,
                    args.smtp_passwd,
                    args.smtp_server,
                    args.smtp_destinations,
                    subject,
                    "\n".join(email_text),
                    attachments
                )
        finally:
            driver.quit()

if __name__ == '__main__':
    main()
