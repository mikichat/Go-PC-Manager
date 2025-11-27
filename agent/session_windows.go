package main

import (
	"fmt"
	"log"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modwtsapi32 = windows.NewLazySystemDLL("wtsapi32.dll")
	modkernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modadvapi32 = windows.NewLazySystemDLL("advapi32.dll")
	moduserenv  = windows.NewLazySystemDLL("userenv.dll")

	procWTSGetActiveConsoleSessionId = modwtsapi32.NewProc("WTSGetActiveConsoleSessionId")
	procWTSQueryUserToken            = modwtsapi32.NewProc("WTSQueryUserToken")
	procDuplicateTokenEx             = modadvapi32.NewProc("DuplicateTokenEx")
	procCreateEnvironmentBlock       = moduserenv.NewProc("CreateEnvironmentBlock")
	procDestroyEnvironmentBlock      = moduserenv.NewProc("DestroyEnvironmentBlock")
	procCreateProcessAsUserW         = modadvapi32.NewProc("CreateProcessAsUserW")
)

func runAsUser(command string) error {
	log.Println("runAsUser: Starting")
	// 1. Get active console session ID
	sessionID, _, _ := procWTSGetActiveConsoleSessionId.Call()
	if sessionID == 0xFFFFFFFF {
		return fmt.Errorf("no active console session found")
	}
	log.Printf("runAsUser: Session ID %d", sessionID)

	// 2. Get user token
	var token windows.Token
	ret, _, err := procWTSQueryUserToken.Call(uintptr(sessionID), uintptr(unsafe.Pointer(&token)))
	if ret == 0 {
		return fmt.Errorf("WTSQueryUserToken failed: %v", err)
	}
	defer token.Close()
	log.Println("runAsUser: Got user token")

	// 3. Duplicate token
	var duplicatedToken windows.Token
	ret, _, err = procDuplicateTokenEx.Call(
		uintptr(token),
		0x02000000, // MAXIMUM_ALLOWED
		0,
		uintptr(windows.SecurityIdentification),
		uintptr(windows.TokenPrimary),
		uintptr(unsafe.Pointer(&duplicatedToken)),
	)
	if ret == 0 {
		return fmt.Errorf("DuplicateTokenEx failed: %v", err)
	}
	defer duplicatedToken.Close()
	log.Println("runAsUser: Duplicated token")

	// 4. Create environment block
	var envBlock uintptr
	ret, _, err = procCreateEnvironmentBlock.Call(
		uintptr(unsafe.Pointer(&envBlock)),
		uintptr(duplicatedToken),
		0, // FALSE (inherit)
	)
	if ret == 0 {
		return fmt.Errorf("CreateEnvironmentBlock failed: %v", err)
	}
	defer procDestroyEnvironmentBlock.Call(envBlock)
	log.Println("runAsUser: Created environment block")

	// 5. Create process as user
	si := windows.StartupInfo{}
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Desktop = windows.StringToUTF16Ptr("winsta0\\default")

	pi := windows.ProcessInformation{}

	cmdLine, err := windows.UTF16PtrFromString(command)
	if err != nil {
		return err
	}

	log.Printf("runAsUser: Calling CreateProcessAsUserW for %s", command)
	// CreateProcessAsUserW arguments:
	// hToken, lpApplicationName, lpCommandLine, lpProcessAttributes, lpThreadAttributes,
	// bInheritHandles, dwCreationFlags, lpEnvironment, lpCurrentDirectory, lpStartupInfo, lpProcessInformation
	ret, _, err = procCreateProcessAsUserW.Call(
		uintptr(duplicatedToken),
		0, // ApplicationName
		uintptr(unsafe.Pointer(cmdLine)),
		0, // ProcessAttributes
		0, // ThreadAttributes
		0, // InheritHandles
		windows.CREATE_UNICODE_ENVIRONMENT|windows.CREATE_NEW_CONSOLE, // CreationFlags
		envBlock,
		0, // CurrentDirectory
		uintptr(unsafe.Pointer(&si)),
		uintptr(unsafe.Pointer(&pi)),
	)
	if ret == 0 {
		return fmt.Errorf("CreateProcessAsUser failed: %v", err)
	}

	windows.CloseHandle(windows.Handle(pi.Process))
	windows.CloseHandle(windows.Handle(pi.Thread))

	log.Printf("Launched GUI process: %s (PID: %d)", command, pi.ProcessId)
	return nil
}
