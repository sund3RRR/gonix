{ outputs, name, system ? builtins.currentSystem }:

let
  hasAttr = attr: value:
    builtins.isAttrs value && builtins.hasAttr attr value;

  getAttr = attr: default: value:
    if hasAttr attr value then builtins.getAttr attr value else default;

  safe = value: default:
    let result = builtins.tryEval value;
    in if result.success then result.value else default;

  asString = value:
    if value == null then ""
    else if builtins.isString value || builtins.isPath value then builtins.toString value
    else "";

  asBool = value:
    if builtins.isBool value then value else false;

  asList = value:
    if builtins.isList value then value
    else if value == null then []
    else [ value ];

  stringList = value:
    builtins.filter (item: item != "") (map asString (asList value));

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
  src = getAttr "src" {} pkg;

  parseSystem = value:
    let
      text = asString value;
      matched = builtins.match "(.+)-([^-]+)" text;
    in {
      system = text;
      arch = if matched == null then "" else builtins.elemAt matched 0;
      os = if matched == null then "" else builtins.elemAt matched 1;
    };

  normalizeLicense = license:
    map (item:
      if builtins.isAttrs item then {
        free = asBool (getAttr "free" false item);
        redistributable = asBool (getAttr "redistributable" false item);
        deprecated = asBool (getAttr "deprecated" false item);
        shortName = asString (getAttr "shortName" "" item);
        fullName = asString (getAttr "fullName" "" item);
        spdxId = asString (getAttr "spdxId" "" item);
        url = asString (getAttr "url" "" item);
      } else {
        free = false;
        redistributable = false;
        deprecated = false;
        shortName = asString item;
        fullName = asString item;
        spdxId = "";
        url = "";
      }
    ) (asList license);

  sourceProvenanceName = item:
    if builtins.isAttrs item then asString (getAttr "shortName" "" item) else asString item;

  sourceHash =
    let hash = asString (getAttr "hash" "" src);
        sha256 = asString (getAttr "sha256" "" src);
    in if hash != "" then hash else sha256;

  outputNames =
    let names = getAttr "outputs" [] pkg;
    in if builtins.isList names && names != [] then names
       else if hasAttr "outPath" pkg then [ "out" ]
       else [];

  outputPath = outputName:
    let value = safe (builtins.getAttr outputName pkg) null;
    in
      if builtins.isAttrs value && hasAttr "outPath" value then asString value.outPath
      else if value != null then asString value
      else if outputName == getAttr "outputName" "out" pkg then asString (getAttr "outPath" "" pkg)
      else "";

  outputAttrs = builtins.listToAttrs (map (outputName: {
    name = outputName;
    value = {
      type = asString (getAttr "type" "" pkg);
      name = asString (getAttr "name" "" pkg);
      drvPath = asString (getAttr "drvPath" "" pkg);
      outPath = outputPath outputName;
      outputName = outputName;
    };
  }) outputNames);

in {
  type = asString (getAttr "type" "" pkg);
  name = asString (getAttr "name" "" pkg);
  pname = asString (getAttr "pname" "" pkg);
  version = asString (getAttr "version" "" pkg);
  system = asString (getAttr "system" system pkg);
  builder = asString (getAttr "builder" "" pkg);
  args = stringList (getAttr "args" [] pkg);
  drvPath = asString (getAttr "drvPath" "" pkg);
  outPath = asString (getAttr "outPath" "" pkg);
  outputName = asString (getAttr "outputName" "" pkg);
  pos = asString (getAttr "pos" (getAttr "position" "" meta) pkg);
  outputs = outputAttrs;

  src = {
    type = asString (getAttr "type" "" src);
    name = asString (getAttr "name" "" src);
    outPath = asString (getAttr "outPath" "" src);
    drvPath = asString (getAttr "drvPath" "" src);
    url = asString (getAttr "url" "" src);
    urls = stringList (getAttr "urls" [] src);
    rev = asString (getAttr "rev" "" src);
    ref = asString (getAttr "ref" "" src);
    owner = asString (getAttr "owner" "" src);
    repo = asString (getAttr "repo" "" src);
    hash = sourceHash;
  };

  meta = {
    broken = asBool (getAttr "broken" false meta);
    unfree = asBool (getAttr "unfree" false meta);
    insecure = asBool (getAttr "insecure" false meta);
    unsupported = asBool (getAttr "unsupported" false meta);
    available = asBool (getAttr "available" false meta);
    description = asString (getAttr "description" "" meta);
    longDescription = asString (getAttr "longDescription" "" meta);
    homepage = asString (getAttr "homepage" "" meta);
    downloadPage = asString (getAttr "downloadPage" "" meta);
    changelog = asString (getAttr "changelog" "" meta);
    mainProgram = asString (getAttr "mainProgram" "" meta);
    position = asString (getAttr "position" "" meta);
    knownVulnerabilities = stringList (getAttr "knownVulnerabilities" [] meta);
    sourceProvenance =
      builtins.filter (item: item != "")
        (map sourceProvenanceName (asList (getAttr "sourceProvenance" [] meta)));
    license = normalizeLicense (getAttr "license" [] meta);
    platforms = map parseSystem (asList (getAttr "platforms" [] meta));
    badPlatforms = map parseSystem (asList (getAttr "badPlatforms" [] meta));
  };
}
