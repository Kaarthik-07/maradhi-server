# shell.nix — backwards compatibility for `nix-shell` (non-flake workflow)
#
# If you're using the older `nix-shell` instead of `nix develop`, this
# delegates to the flake so you don't maintain two separate shell definitions.
#
# Usage:
#   nix-shell          (uses this file, which delegates to flake.nix)
#   nix develop        (uses flake.nix directly — preferred)
#
# Both produce the identical environment.
(import
  (
    let
      # Pin the flake-compat version — ensures nix-shell also gets a
      # reproducible environment matching flake.lock
      lock = builtins.fromJSON (builtins.readFile ./flake.lock);
      compat = lock.nodes.flake-compat or null;
    in
    if compat != null
    then
      fetchTarball {
        url    = "https://github.com/edolstra/flake-compat/archive/${compat.locked.rev}.tar.gz";
        sha256 = compat.locked.narHash;
      }
    else
      # Fallback: fetch flake-compat directly if not in lock yet
      fetchTarball "https://github.com/edolstra/flake-compat/archive/master.tar.gz"
  )
  { src = ./.; }
).shellNix
