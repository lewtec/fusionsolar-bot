{ lib
, buildGoModule
, chromium
, makeWrapper
}:

buildGoModule {
  pname = "fusionsolar-bot";
  version = builtins.readFile ./version.txt;

  src = ./.;

  vendorHash = "sha256-etxK2HPE+X5bsYyQKNRpvb4BLzP9JW+0bxpPQocgrBg=";

  nativeBuildInputs = [ makeWrapper ];

  postInstall = ''
    wrapProgram $out/bin/fusionsolar-bot \
      --prefix PATH : ${lib.makeBinPath [ chromium ]}
  '';

  meta = {
    description = "A bot for interacting with FusionSolar";
    homepage = "https://github.com/username/fusionsolar-bot";
    license = lib.licenses.mit;
    mainProgram = "fusionsolar-bot";
    maintainers = with lib.maintainers; [ lucasew ];
  };
}
