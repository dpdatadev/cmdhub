package internal

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/google/uuid"
)

var (
	PrintIdentity = color.New(color.Bold, color.FgGreen, color.Italic).PrintfFunc()
	PrintSuccess  = color.New(color.Bold, color.FgGreen, color.Underline).PrintfFunc()
	PrintStdOut   = color.New(color.Bold, color.FgYellow).PrintfFunc()
	PrintStdErr   = color.New(color.Bold, color.FgHiRed).PrintfFunc()
	PrintFailure  = color.New(color.Bold, color.FgRed, color.Underline).PrintfFunc()
	PrintDebug    = color.New(color.Bold, color.FgBlue, color.Italic).PrintfFunc()
)

type CmdIOHelper struct{}

func (io *CmdIOHelper) ParseHubCommands(fileName string) []*HubCommand {

	PrintDebug("HubCommand READ[+]: %s\n", fileName)

	fileName = strings.ToLower(strings.TrimSpace(fileName))

	//Check file extension (replace with YAML in BETA)
	if !strings.HasSuffix(fileName, ".txt") {
		PrintFailure("Invalid file type: %s\n", fileName)
		log.Println("Only .TXT files supported at this time for parsing (alpha v0.1)")
		return []*HubCommand{}
	}

	file := io.GetFileRead(fileName)
	//Handle file open
	if file == nil {
		PrintFailure("Error opening file: %v\n", errors.New("file is nil"))
		return []*HubCommand{}
	}

	defer file.Close()

	// Process the file
	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	if err != nil {
		PrintFailure("Error reading file: %v\n", err)
		return []*HubCommand{}
	}
	HubCommandData := string(buf[:n])
	HubCommands := make([]*HubCommand, 0, len(HubCommandData))
	HubCommandLines := strings.SplitSeq(HubCommandData, "\n")
	for cmd := range HubCommandLines {
		//TODO, eventually handling TOML or YAML or Proc files, not plain .txt
		//ignore commented out HubCommands
		if !strings.HasPrefix(cmd, "//") && !strings.HasPrefix(cmd, "##") { //TODO, test coverage
			cmdFields := strings.Fields(cmd)
			cmdName := cmdFields[0]
			cmdArgs := cmdFields[1:]
			cmdNotes := fmt.Sprintf("Ingested from %s", fileName)
			HubCommand := NewHubCommand(cmdName, cmdArgs, cmdNotes)
			HubCommands = append(HubCommands, HubCommand)
			PrintDebug("Ingested HubCommand: %s, Args: %v\n", cmdName, cmdArgs)
		}
	}

	return HubCommands
}

// remove in beta
func (io *CmdIOHelper) getHostIpConfig() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}

	var ipAddrStr string

	var IPV4or6 string
	var IPError error

	for _, addr := range addrs {
		// Check the address type
		if ipnet, ok := addr.(*net.IPNet); ok {
			// See if it's a valid IPV4 loopback
			/*if ipnet.IP.To4() != nil && ipnet.IP.IsLoopback() {
				IPV4or6 = ipnet.IP.String()
				IPError = nil
				ipAddrStr = fmt.Sprintf("Loopback: %s\n", IPV4or6)
			}*/
			// Check if IPV4 or IPV6 and assign to builder
			if ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
				IPV4or6 = ipnet.IP.String()
				IPError = nil
				ipAddrStr = fmt.Sprintf("Host IPv4 Address: %s\n", IPV4or6)
			} else if ipnet.IP.To16() != nil && !ipnet.IP.IsLoopback() {
				IPV4or6 = ipnet.IP.String()
				IPError = nil
				ipAddrStr = fmt.Sprintf("Host IPv6 Address: %s\n", IPV4or6)
			}

		} else { // Finally, no valid addresses and/or something is !ok
			IPError = errors.New("ERR - IP RETRIEVAL")
			return string(""), IPError
		}
	}
	return ipAddrStr, IPError
}

// ANSI SQL LEFT style substring
func (io *CmdIOHelper) Left(s string, size int) (string, error) {

	if s == "" {
		return s, errors.New("EMPTY STRING")
	}

	leftSubstr := s[:size]

	return leftSubstr, nil
}

// ANSI SQL RIGHT style substring
func (io *CmdIOHelper) Right(s string, size int) (string, error) {
	if s == "" {
		return s, errors.New("EMPTY STRING")
	}

	appliedSize := max((len(s) - size), 0)

	return s[appliedSize:], nil
}

// Return files for Logging or dumping
func (io *CmdIOHelper) GetFileWrite(fileName string) *os.File {
	if fileName == "" {
		PrintFailure("errors.New(\"\"): %v\n", errors.New("WRITE FILE ERROR"))
		return nil
	}

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		PrintFailure("errors.New(\"\"): %v\n", err)
		return nil
	}

	return file
}

func (io *CmdIOHelper) GetFileRead(fileName string) *os.File {
	if fileName == "" {
		PrintFailure("errors.New(\"\"): %v\n", errors.New("READ FILE ERROR"))
		return nil
	}

	file, err := os.Open(fileName)
	if err != nil {
		PrintFailure("errors.New(\"\"): %v\n", err)
		return nil
	}

	return file
}

// Version 4 Google UUID (length 7) (UNSAFE, INTERNAL USE ONLY (lineage/testing))
func (io *CmdIOHelper) NewShortUUID() (string, error) {

	uuidString, err := io.Left(uuid.NewString(), 8)

	return uuidString, err
}

// Helper function for displaying/dumping HubCommand info (default Console/Text/Printf())
func (io *CmdIOHelper) ConsoleDump(cmd *HubCommand) {
	if cmd.Stderr != "" || cmd.Status == "FAILED" {
		PrintFailure("HubCommand ID: %v\n", cmd.ID)
		PrintFailure("HubCommand Name: %s\n", cmd.Name)
		PrintFailure("HubCommand Args: %s\n", cmd.Args)
		PrintFailure("Status: %v\n", cmd.Status)
		PrintStdErr("STDERR: %s::<%s>\n", cmd.Stderr, cmd.Error)
		//ConsoleStdErrHandle(cmd.Stderr) //TODO
	} else if cmd.Status == "SUCCESS" {
		PrintIdentity("\nHubCommand ID: %v\n", cmd.ID)
		PrintIdentity("HubCommand Name: %s\n", cmd.Name)
		PrintIdentity("HubCommand Args: %s\n", cmd.Args)
		PrintSuccess("Status: %v\n", cmd.Status)
		PrintStdOut("STDOUT:\n %s\n", cmd.Stdout)
		fmt.Println()
		//ConsoleStdOutHandle(cmd.Stdout) //TODO
	} else {
		fmt.Println(fmt.Errorf("UNKNOWN ERROR OCCURRED: %s", cmd.ID.String()))
	}
}

func (io *CmdIOHelper) FileDump(cmd *HubCommand, logFileName string) {

	logFile := io.GetFileWrite(logFileName)

	if logFile == nil {
		PrintFailure("errors.New(\"\"): %v\n", errors.New("FILE ERROR"))
		return
	}

	defer logFile.Close()
	log.SetOutput(logFile)

	if cmd.Stderr != "" || cmd.Status == "FAILED" {
		log.Fatalf("HubCommand ID: %v\n", cmd.ID)
		log.Fatalf("HubCommand Name: %s\n", cmd.Name)
		log.Fatalf("HubCommand Args: %s\n", cmd.Args)
		log.Fatalf("Status: %v\n", cmd.Status)
		log.Fatalf("STDERR: %s::<%s>\n", cmd.Stderr, cmd.Error)
		//ConsoleStdErrHandle(cmd.Stderr) //TODO
	} else if cmd.Status == "SUCCESS" {
		log.Printf("HubCommand ID: %v\n", cmd.ID)
		log.Printf("HubCommand Name: %s\n", cmd.Name)
		log.Printf("HubCommand Args: %s\n", cmd.Args)
		log.Printf("Status: %v\n", cmd.Status)
		log.Printf("STDOUT:\n %s\n", cmd.Stdout)
		//ConsoleStdOutHandle(cmd.Stdout) //TODO
	} else {
		PrintFailure("UNKNOWN ERROR OCCURRED: %s\n", cmd.ID.String())
	}
}
