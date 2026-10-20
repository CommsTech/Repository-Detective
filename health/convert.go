package health

// FileContent mirrors analyzers.FileContent without an import cycle.
type FileContent FileInput

// InputsFromFileContents converts analyzer file contents to health inputs.
func InputsFromFileContents(files []FileContent) []FileInput {
	out := make([]FileInput, 0, len(files))
	for _, f := range files {
		out = append(out, FileInput(f))
	}
	return out
}
