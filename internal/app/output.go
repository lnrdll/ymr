package app

import "github.com/lnrdll/ymr/internal/processor"

func handleOutput(outputPort OutputPort, outputs []processor.RenderedOutput, terminalOutput bool, outputDir string) error {
	return outputPort.Write(outputs, terminalOutput, outputDir)
}

func prepareOutputDir(cfgOutputDir string) (string, bool) {
	if cfgOutputDir == "-" {
		return "", true
	}

	return cfgOutputDir, false
}
