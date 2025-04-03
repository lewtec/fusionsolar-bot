{ chromedriver
, chromium
, selenium
, sentry-sdk
, hatchling
, buildPythonPackage
, lib
, python3
}:

buildPythonPackage {
  pname = "fusionsolar-bot";
  version = builtins.readFile ./version.txt;
  pyproject = true;

  src = ./.;

  build-system = [ hatchling ];
  dependencies = [ selenium sentry-sdk ];

  makeWrapperArgs = [
    "--prefix" "PATH" ":" "${lib.makeBinPath [ chromedriver chromium ]}"
  ];

  # Add proper checkInputs if there are tests
  checkInputs = [ python3.pkgs.pytest ];
  pythonImportsCheck = [ "fusionsolar_bot" ];

  meta = {
    description = "A bot for interacting with FusionSolar";
    homepage = "https://github.com/username/fusionsolar-bot";
    license = lib.licenses.mit;
    mainProgram = "fusionsolar-bot";
    maintainers = with lib.maintainers; [ lucasew ];
  };
}
