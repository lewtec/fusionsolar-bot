{
  config,
  lib,
  ...
}:

let
  inherit (lib)
    mkOption
    mkEnableOption
    mkForce
    types
    mkIf
    ;
  cfg = config.services.fusionsolar-reporter;
in

{
  options.services.fusionsolar-reporter = {
    enable = mkEnableOption "fusionsolar-reporter";

    environmentFile = mkOption {
      type = types.path;
      example = "/var/run/secrets/fusionsolar";
      default = "/var/run/secrets/fusionsolar";
      description = "Credentials file";
    };

    image = mkOption {
      description = "Which cf-torrent image to use";
      default = "ghcr.io/lucasew/fusionsolar-bot:latest";
      type = types.str;
    };

    calendar = mkOption {
      type = types.str;
      default = "20:00:01";
      description = "When to run the report";
    };
  };

  config = mkIf cfg.enable {
    systemd.timers.fusionsolar-reporter = {
      description = "Fusionsolar reporter timer";
      wantedBy = [ "timers.target" ];
      timerConfig = {
        OnCalendar = cfg.calendar;
        AccuracySec = "30m";
        Unit = "fusionsolar-reporter.service";
      };
    };

    virtualisation.oci-containers.containers.fusionsolar-reporter = {
      inherit (cfg) image;
      environmentFiles = [ cfg.environmentFile ];
      pull = "always";
      serviceName = "fusionsolar-reporter";
      autoStart = false;
    };

    systemd.services.fusionsolar-reporter.serviceConfig.Restart = mkForce "no";
  };
}
