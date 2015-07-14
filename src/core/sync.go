package core

// Sync synchronizes files (f) to device. Sync
// will copy new files, delete old files, and fix or update files on the
// destination drive that do not match the source sha1 hash.
func Sync(f fileList, d DriveList, destPaths []string) {
	for _, y := range *f {
		log.Debug("test", "y", y)
	}
	// drive = backupDrives.AvailableSpace(drive, x)
	// mIo := NewIoReaderWriter(oFile, x.Size)
	// logs.Printf("Writing %q (%s)...\n\r",
	// x.Name, humanize.IBytes(x.Size))
	// nIo := mIo.MultiWriter()
	// if oSize, err := io.Copy(nIo, sFile); err != nil {
	// return err
	// } else {
	// backupDrives[drive].UsedSize += uint64(oSize)
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
