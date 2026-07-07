package gonix

// RealizedOutput describes one realized derivation output.
//
// It contains only Go-owned data. Client.Realize and Client.RealizeOutput
// close the underlying Nix store path handles before returning this DTO.
type RealizedOutput struct {
	OutputName string   `json:"outputName"`
	StorePath  string   `json:"storePath"`
	RealPath   string   `json:"realPath"`
	Name       string   `json:"name"`
	Hash       [20]byte `json:"hash"`
}
