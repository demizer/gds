package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/demizer/go-humanize"
)

// Sync synchronizes files (f) to mounted devices (d) at mountpoints
// (destPaths). Sync will copy new files, delete old files, and fix or update
// files on the destination device that do not match the source sha1 hash.
func Sync(f FileList, d DeviceList, destPaths []string) error {
	var device int
	for _, y := range f {
		fName := filepath.Join(destPaths[device], y.Name)
		// Open source file for reading
		sFile, err := os.Open(y.Path)
		defer sFile.Close()
		if err != nil {
			return fmt.Errorf("Could not open source file %q for writing! -- %s", y.Path, err)
		}
		var oFile io.Writer
		// Open dest file for writing
		if destPaths[device] == "/dev/null" {
			oFile = ioutil.Discard
		} else {
			oFile, err := os.Open(fName)
			defer oFile.Close()
			if err != nil {
				return fmt.Errorf("Could not open dest file %q for writing! -- %q", fName, err)
			}
		}
		mIo := NewIoReaderWriter(oFile, y.Size)
		fmt.Printf("Writing %q (%s)...\n\r", y.Name, humanize.IBytes(y.Size))
		nIo := mIo.MultiWriter()
		if oSize, err := io.Copy(nIo, sFile); err != nil {
			return fmt.Errorf("Error writing file %q, err: %q", fName, err)
		} else {
			d[device].UsedSize += uint64(oSize)
		}
		fmt.Println("")
		sFile.Close()
	}
	// device = backupDevices.AvailableSpace(device, x)
	// x.SrcSha = mIo.Sha1SumToString()
	// var vDestShaErr error
	// fmt.Println("\r")
	// Println("Verifying hash of destination file...")
	// x.DestSha, vDestShaErr = verifyDestinationFileSHA(x)
	// if err := writeJson(&x); err != nil {
	// return err
	// }
	// if vDestShaErr != nil {
	// return err
	// } else {
	// Println("Destination hash is good! :)")
	// }
	// }
	return nil
}
