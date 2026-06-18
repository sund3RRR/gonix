package gonix

// PackageType identifies the broad kind of a package-shaped value.
type PackageType string

const (
	// PackageTypeDerivation identifies a derivation-like package value.
	PackageTypeDerivation PackageType = "derivation"
	// PackageTypeApp identifies a flake app value.
	PackageTypeApp PackageType = "app"
)

// SourceType identifies the broad kind of a source-shaped value.
type SourceType string

const (
	// SourceTypePath identifies a path source value.
	SourceTypePath SourceType = "path"
	// SourceTypeGit identifies a Git source value.
	SourceTypeGit SourceType = "git"
	// SourceTypeGitHub identifies a GitHub source value.
	SourceTypeGitHub SourceType = "github"
	// SourceTypeGitLab identifies a GitLab source value.
	SourceTypeGitLab SourceType = "gitlab"
	// SourceTypeSourceHut identifies a SourceHut source value.
	SourceTypeSourceHut SourceType = "sourcehut"
	// SourceTypeMercurial identifies a Mercurial source value.
	SourceTypeMercurial SourceType = "mercurial"
	// SourceTypeTarball identifies a tarball source value.
	SourceTypeTarball SourceType = "tarball"
	// SourceTypeFile identifies a single-file source value.
	SourceTypeFile SourceType = "file"
	// SourceTypeURL identifies a URL source value.
	SourceTypeURL SourceType = "url"
	// SourceTypeIndirect identifies an indirect flake reference value.
	SourceTypeIndirect SourceType = "indirect"
)

// Package describes the common public shape of an evaluated Nix package.
//
// In Nix this is normally a derivation-like attribute set, not a source file.
// The source expression, such as package.nix or default.nix, evaluates to this
// value. drvPath points to the generated .drv build plan and outPath points to
// the default realized output path.
type Package struct {
	Type       PackageType              `nix:"type" json:"type" validate:"optional"`
	Name       string                   `nix:"name" json:"name" validate:"optional"`
	PName      string                   `nix:"pname" json:"pname" validate:"optional"`
	Version    string                   `nix:"version" json:"version" validate:"optional"`
	System     string                   `nix:"system" json:"system" validate:"optional"`
	Builder    string                   `nix:"builder" json:"builder" validate:"optional"`
	Args       []string                 `nix:"args" json:"args" validate:"optional"`
	DrvPath    string                   `nix:"drvPath" json:"drvPath" validate:"optional"`
	OutPath    string                   `nix:"outPath" json:"outPath" validate:"optional"`
	OutputName string                   `nix:"outputName" json:"outputName" validate:"optional"`
	Position   string                   `nix:"pos" json:"pos" validate:"optional"`
	Src        Source                   `nix:"src" json:"src" validate:"optional"`
	Meta       PackageMeta              `nix:"meta" json:"meta" validate:"optional"`
	Outputs    map[string]PackageOutput `nix:"outputs" json:"outputs" validate:"required"`
}

// PackageOutput describes a named output attribute of an evaluated package.
type PackageOutput struct {
	Type       PackageType `nix:"type" json:"type" validate:"optional"`
	Name       string      `nix:"name" json:"name" validate:"optional"`
	DrvPath    string      `nix:"drvPath" json:"drvPath" validate:"optional"`
	OutPath    string      `nix:"outPath" json:"outPath" validate:"optional"`
	OutputName string      `nix:"outputName" json:"outputName" validate:"optional"`
}

// RealizedPackageOutput describes one realized package output.
//
// It contains only Go-owned data. RealizePackage closes the underlying Nix
// store path handles before returning this DTO.
type RealizedPackageOutput struct {
	OutputName string   `json:"outputName"`
	StorePath  string   `json:"storePath"`
	RealPath   string   `json:"realPath"`
	Name       string   `json:"name"`
	Hash       [20]byte `json:"hash"`
}

// PackageMeta describes the conventional nixpkgs meta attribute set.
//
// Many meta fields are intentionally optional because nixpkgs does not require
// every package to define them. The package projection normalizes richer
// nixpkgs metadata into this stable Go-native surface.
type PackageMeta struct {
	Broken               bool         `nix:"broken" json:"broken" validate:"optional"`
	Unfree               bool         `nix:"unfree" json:"unfree" validate:"optional"`
	Insecure             bool         `nix:"insecure" json:"insecure" validate:"optional"`
	Unsupported          bool         `nix:"unsupported" json:"unsupported" validate:"optional"`
	Available            bool         `nix:"available" json:"available" validate:"optional"`
	Description          string       `nix:"description" json:"description" validate:"optional"`
	LongDescription      string       `nix:"longDescription" json:"longDescription" validate:"optional"`
	Homepage             string       `nix:"homepage" json:"homepage" validate:"optional"`
	DownloadPage         string       `nix:"downloadPage" json:"downloadPage" validate:"optional"`
	Changelog            string       `nix:"changelog" json:"changelog" validate:"optional"`
	MainProgram          string       `nix:"mainProgram" json:"mainProgram" validate:"optional"`
	Position             string       `nix:"position" json:"position" validate:"optional"`
	KnownVulnerabilities []string     `nix:"knownVulnerabilities" json:"knownVulnerabilities" validate:"optional"`
	SourceProvenance     []string     `nix:"sourceProvenance" json:"sourceProvenance" validate:"optional"`
	License              []License    `nix:"license" json:"license" validate:"optional"`
	Maintainers          []Maintainer `nix:"maintainers" json:"maintainers" validate:"optional"`
	Platforms            []Platform   `nix:"platforms" json:"platforms" validate:"optional"`
	BadPlatforms         []Platform   `nix:"badPlatforms" json:"badPlatforms" validate:"optional"`
}

// License describes a nixpkgs lib.licenses entry.
type License struct {
	Free            bool   `nix:"free" json:"free" validate:"optional"`
	Redistributable bool   `nix:"redistributable" json:"redistributable" validate:"optional"`
	Deprecated      bool   `nix:"deprecated" json:"deprecated" validate:"optional"`
	ShortName       string `nix:"shortName" json:"shortName" validate:"optional"`
	FullName        string `nix:"fullName" json:"fullName" validate:"optional"`
	SpdxID          string `nix:"spdxId" json:"spdxId" validate:"optional"`
	URL             string `nix:"url" json:"url" validate:"optional"`
}

// Platform describes a parsed nixpkgs platform system entry.
type Platform struct {
	System string `nix:"system" json:"system" validate:"optional"`
	Arch   Arch   `nix:"arch" json:"arch" validate:"optional"`
	OS     OS     `nix:"os" json:"os" validate:"optional"`
}

// Maintainer describes a nixpkgs lib.maintainers entry.
type Maintainer struct {
	Name   string          `nix:"name" json:"name" validate:"optional"`
	Email  string          `nix:"email" json:"email" validate:"optional"`
	GitHub string          `nix:"github" json:"github" validate:"optional"`
	GitLab string          `nix:"gitlab" json:"gitlab" validate:"optional"`
	Matrix string          `nix:"matrix" json:"matrix" validate:"optional"`
	Keys   []MaintainerKey `nix:"keys" json:"keys" validate:"optional"`
}

// MaintainerKey describes a nixpkgs maintainer OpenPGP key entry.
type MaintainerKey struct {
	Fingerprint string `nix:"fingerprint" json:"fingerprint" validate:"optional"`
	LongKeyID   string `nix:"longkeyid" json:"longkeyid" validate:"optional"`
}

// Source describes the common source fetcher result shape attached to src.
type Source struct {
	Type    SourceType `nix:"type" json:"type" validate:"optional"`
	Name    string     `nix:"name" json:"name" validate:"optional"`
	OutPath string     `nix:"outPath" json:"outPath" validate:"optional"`
	DrvPath string     `nix:"drvPath" json:"drvPath" validate:"optional"`
	URL     string     `nix:"url" json:"url" validate:"optional"`
	Rev     string     `nix:"rev" json:"rev" validate:"optional"`
	Ref     string     `nix:"ref" json:"ref" validate:"optional"`
	Owner   string     `nix:"owner" json:"owner" validate:"optional"`
	Repo    string     `nix:"repo" json:"repo" validate:"optional"`
	Hash    string     `nix:"hash" json:"hash" validate:"optional"`
	URLs    []string   `nix:"urls" json:"urls" validate:"optional"`
}
