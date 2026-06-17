package gonix

// PackageType identifies the broad kind of a package-shaped value.
type PackageType string

const (
	// PackageTypeSource identifies a source-like package value.
	PackageTypeSource PackageType = "source"
	// PackageTypeDerivation identifies a derivation-like package value.
	PackageTypeDerivation PackageType = "derivation"
	// PackageTypeApp identifies a flake app value.
	PackageTypeApp PackageType = "app"
	// PackageTypePath identifies a path source value.
	PackageTypePath PackageType = "path"
	// PackageTypeGit identifies a Git source value.
	PackageTypeGit PackageType = "git"
	// PackageTypeGitHub identifies a GitHub source value.
	PackageTypeGitHub PackageType = "github"
	// PackageTypeGitLab identifies a GitLab source value.
	PackageTypeGitLab PackageType = "gitlab"
	// PackageTypeSourceHut identifies a SourceHut source value.
	PackageTypeSourceHut PackageType = "sourcehut"
	// PackageTypeMercurial identifies a Mercurial source value.
	PackageTypeMercurial PackageType = "mercurial"
	// PackageTypeTarball identifies a tarball source value.
	PackageTypeTarball PackageType = "tarball"
	// PackageTypeFile identifies a single-file source value.
	PackageTypeFile PackageType = "file"
	// PackageTypeURL identifies a URL source value.
	PackageTypeURL PackageType = "url"
	// PackageTypeIndirect identifies an indirect flake reference value.
	PackageTypeIndirect PackageType = "indirect"
)

// Package describes the common public shape of an evaluated Nix package.
//
// In Nix this is normally a derivation-like attribute set, not a source file.
// The source expression, such as package.nix or default.nix, evaluates to this
// value. drvPath points to the generated .drv build plan and outPath points to
// the default realized output path.
type Package struct {
	Type       PackageType              `nix:"type" validate:"optional"`
	Name       string                   `nix:"name" validate:"optional"`
	PName      string                   `nix:"pname" validate:"optional"`
	Version    string                   `nix:"version" validate:"optional"`
	System     string                   `nix:"system" validate:"optional"`
	Builder    string                   `nix:"builder" validate:"optional"`
	Args       []string                 `nix:"args" validate:"optional"`
	DrvPath    string                   `nix:"drvPath" validate:"optional"`
	OutPath    string                   `nix:"outPath" validate:"optional"`
	OutputName string                   `nix:"outputName" validate:"optional"`
	Position   string                   `nix:"pos" validate:"optional"`
	Src        Source                   `nix:"src" validate:"optional"`
	Meta       PackageMeta              `nix:"meta" validate:"optional"`
	Outputs    map[string]PackageOutput `nix:"outputs" validate:"required"`
}

// System describes a parsed Nix system string.
type System struct {
	Name string `nix:"name" validate:"optional"`
	Arch Arch   `nix:"arch" validate:"optional"`
	OS   OS     `nix:"os" validate:"optional"`
}

// PackageOutput describes a named output attribute of an evaluated package.
type PackageOutput struct {
	Type       PackageType `nix:"type" validate:"optional"`
	Name       string      `nix:"name" validate:"optional"`
	DrvPath    string      `nix:"drvPath" validate:"optional"`
	OutPath    string      `nix:"outPath" validate:"optional"`
	OutputName string      `nix:"outputName" validate:"optional"`
}

// PackageMeta describes the conventional nixpkgs meta attribute set.
//
// Many meta fields are intentionally optional because nixpkgs does not require
// every package to define them. The package projection normalizes richer
// nixpkgs metadata into this stable Go-native surface.
type PackageMeta struct {
	Broken               bool         `nix:"broken" validate:"optional"`
	Unfree               bool         `nix:"unfree" validate:"optional"`
	Insecure             bool         `nix:"insecure" validate:"optional"`
	Unsupported          bool         `nix:"unsupported" validate:"optional"`
	Available            bool         `nix:"available" validate:"optional"`
	Description          string       `nix:"description" validate:"optional"`
	LongDescription      string       `nix:"longDescription" validate:"optional"`
	Homepage             string       `nix:"homepage" validate:"optional"`
	DownloadPage         string       `nix:"downloadPage" validate:"optional"`
	Changelog            string       `nix:"changelog" validate:"optional"`
	MainProgram          string       `nix:"mainProgram" validate:"optional"`
	Position             string       `nix:"position" validate:"optional"`
	KnownVulnerabilities []string     `nix:"knownVulnerabilities" validate:"optional"`
	SourceProvenance     []string     `nix:"sourceProvenance" validate:"optional"`
	License              []License    `nix:"license" validate:"optional"`
	Maintainers          []Maintainer `nix:"maintainers" validate:"optional"`
	Platforms            []Platform   `nix:"platforms" validate:"optional"`
	BadPlatforms         []Platform   `nix:"badPlatforms" validate:"optional"`
}

// License describes a nixpkgs lib.licenses entry.
type License struct {
	Free            bool   `nix:"free" validate:"optional"`
	Redistributable bool   `nix:"redistributable" validate:"optional"`
	Deprecated      bool   `nix:"deprecated" validate:"optional"`
	ShortName       string `nix:"shortName" validate:"optional"`
	FullName        string `nix:"fullName" validate:"optional"`
	SpdxID          string `nix:"spdxId" validate:"optional"`
	URL             string `nix:"url" validate:"optional"`
}

// Platform describes a parsed nixpkgs platform system entry.
type Platform struct {
	Name   string `nix:"name" validate:"optional"`
	System string `nix:"system" validate:"optional"`
	Arch   Arch   `nix:"arch" validate:"optional"`
	OS     OS     `nix:"os" validate:"optional"`
}

// Maintainer describes a nixpkgs lib.maintainers entry.
type Maintainer struct {
	Name   string          `nix:"name" validate:"optional"`
	Email  string          `nix:"email" validate:"optional"`
	GitHub string          `nix:"github" validate:"optional"`
	GitLab string          `nix:"gitlab" validate:"optional"`
	Matrix string          `nix:"matrix" validate:"optional"`
	Keys   []MaintainerKey `nix:"keys" validate:"optional"`
}

// MaintainerKey describes a nixpkgs maintainer OpenPGP key entry.
type MaintainerKey struct {
	Fingerprint string `nix:"fingerprint" validate:"optional"`
	LongKeyID   string `nix:"longkeyid" validate:"optional"`
}

// Source describes the common source fetcher result shape attached to src.
type Source struct {
	Type    string   `nix:"type" validate:"optional"`
	Name    string   `nix:"name" validate:"optional"`
	OutPath string   `nix:"outPath" validate:"optional"`
	DrvPath string   `nix:"drvPath" validate:"optional"`
	URL     string   `nix:"url" validate:"optional"`
	Rev     string   `nix:"rev" validate:"optional"`
	Ref     string   `nix:"ref" validate:"optional"`
	Owner   string   `nix:"owner" validate:"optional"`
	Repo    string   `nix:"repo" validate:"optional"`
	Hash    string   `nix:"hash" validate:"optional"`
	URLs    []string `nix:"urls" validate:"optional"`
	Sha256  string   `nix:"sha256" validate:"optional"`
}
