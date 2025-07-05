{
  pkgs ? import <nixpkgs> { },
}:
pkgs.buildGoLatestModule {
  pname = "cage";
  version = "0.0.1";
  src = ./.;
  vendorHash = "sha256-3Q6UJQxuJwXhJorwPcvG5ykZbOjJSC2EZ8WG5xftU64=";
}
