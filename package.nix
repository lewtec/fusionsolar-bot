{ chromedriver
, chromium
, selenium
, sentry-sdk
, hatchling
, buildPythonPackage
}:

buildPythonPackage {
  pname = "fusionsolar-bot";
  version = builtins.readFile ./version.txt;
  pyproject = true;

  src = ./.;

  makeWrapperArgs = [
    "--prefix" "PATH" ":" "${chromedriver}/bin"
    "--prefix" "PATH" ":" "${chromium}/bin"
  ];

  build-system = [ hatchling ];

  dependencies = [ selenium sentry-sdk ];

  meta.mainProgram = "fusionsolar-bot";
}

