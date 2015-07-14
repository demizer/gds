package core

import (
	log "gopkg.in/inconshreveable/log15.v2"
)

// Sync synchronizes files (f) to device. Sync
// will copy new files, delete old files, and fix or update files on the
// destination device that do not match the source sha1 hash.
func Sync(f FileList, d DeviceList, destPaths []string) {
	for _, y := range f {
		log.Debug("test", "y", y)
	}
	// device = backupDevices.AvailableSpace(device, x)
	// mIo := NewIoReaderWriter(oFile, x.Size)
	// logs.Printf("Writing %q (%s)...\n\r",
	// x.Name, humanize.IBytes(x.Size))
	// nIo := mIo.MultiWriter()
	// if oSize, err := io.Copy(nIo, sFile); err != nil {
	// return err
	// } else {
	// backupDevices[device].UsedSize += uint64(oSize)
	// }
	// sFile.Close()
	// x.SrcSha = mIo.Sha256SumToString()
	// var vDestShaErr error
	// fmt.Println("\r")
	// logs.Println("Verifying hash of destination file...")
	// x.DestSha, vDestShaErr = verifyDestinationFileSHA(x)
	// if err := writeJson(&x); err != nil {
	// return err
	// }
	// if vDestShaErr != nil {
	// return err
	// } else {
	// logs.Println("Destination hash is good! :)")
	// }
	// }
}
