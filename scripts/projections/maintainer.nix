{ outputs, name, system ? builtins.currentSystem }:

let
  hasAttr = attr: value:
    builtins.isAttrs value && builtins.hasAttr attr value;

  getAttr = attr: default: value:
    if hasAttr attr value then builtins.getAttr attr value else default;

  asString = value:
    if value == null then ""
    else if builtins.isString value || builtins.isPath value then builtins.toString value
    else "";

  asList = value:
    if builtins.isList value then value
    else if value == null then []
    else [ value ];

  splitName = value:
    builtins.filter (part: builtins.isString part && part != "") (builtins.split "\\." value);

  resolveAttrPath = root: parts:
    if parts == [] then root
    else
      let
        head = builtins.head parts;
        tail = builtins.tail parts;
      in
        if hasAttr head root
        then resolveAttrPath (builtins.getAttr head root) tail
        else throw "package not found: ${name}";

  packageSet =
    if hasAttr "packages" outputs && hasAttr system outputs.packages
    then outputs.packages.${system}
    else throw "flake does not expose packages for system ${system}";

  pkg = resolveAttrPath packageSet (splitName name);
  meta = getAttr "meta" {} pkg;

  normalizeMaintainer = maintainer:
    if builtins.isAttrs maintainer then {
      name = asString (getAttr "name" "" maintainer);
      email = asString (getAttr "email" "" maintainer);
      github = asString (getAttr "github" "" maintainer);
      gitlab = asString (getAttr "gitlab" "" maintainer);
      matrix = asString (getAttr "matrix" "" maintainer);
      keys = map (key: {
        fingerprint = asString (getAttr "fingerprint" "" key);
        longkeyid = asString (getAttr "longkeyid" "" key);
      }) (asList (getAttr "keys" [] maintainer));
    } else {
      name = asString maintainer;
      email = "";
      github = "";
      gitlab = "";
      matrix = "";
      keys = [];
    };

in map normalizeMaintainer (asList (getAttr "maintainers" [] meta))
