{ lib
, buildGoModule
, chromium
, makeWrapper
}:

buildGoModule {
  pname = "fusionsolar-bot";
  version = builtins.readFile ./version.txt;

  src = ./.;

  # vendorHash = null; # Use this if you don't want to enforce vendor consistency, or use specific hash
  # I cannot generate the hash without internet access in the nix build environment.
  # The user should update this hash by running `nix build` and copying the expected hash.
  vendorHash = null;

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
