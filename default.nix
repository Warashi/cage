{
  pkgs ? import <nixpkgs> { },
}:
pkgs.buildGoLatestModule {
  pname = "cage";
  version = "0.1.13";
  src = ./.;
  vendorHash = "sha256-EnEy9KELRFyM+uB1h9mCxuDeUirFiuoLnHURkg8/oQs=";
  meta = {
    mainProgram = "cage";
  };
}
