{ dockerTools
, python3Packages
, lib
, self
, fontconfig
, default
}:

let
  name = "ghcr.io/lucasew/fusionsolar-bot";
  tag = default.version;

  user = {
    name = "user";
    uid = 1000;
    gid = 1000;
  };
in

dockerTools.streamLayeredImage {
  inherit name tag;
  maxLayers = 2;

  contents = [
    dockerTools.binSh
    (dockerTools.fakeNss.override {
      extraPasswdLines = ["${user.name}:x:${toString user.uid}:${toString user.gid}:new user:/tmp:/bin/sh"];
      extraGroupLines = ["${user.name}:x:${toString user.gid}:"];
    })
  ];

  extraCommands = ''
    mkdir -m777 -p tmp etc dev/shm
  '';

  inherit (user) uid gid;
  uname = user.name;
  gname = user.name;

  config = {
    Entrypoint = [
      (lib.getExe default)
      "--headless"
    ];
    User = user.name;
    Env = [
      "HOME=/tmp"
      "LANGUAGE=en_US"
      "UID=${toString user.uid}"
      "GID=${toString user.gid}"
      "TZ=UTC"
      "FONTCONFIG_FILE=${fontconfig.out}/etc/fonts/fonts.conf"
      "FONTCONFIG_PATH=${fontconfig.out}/etc/fonts/"
    ];
  };

  passthru = {
    inherit name tag;
  };
}
