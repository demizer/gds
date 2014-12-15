======================================
todrives -- The External Backup Thingy
======================================

todrives copies files from one large storage pool, to multiple miscellaneous
drives mounted to a single mount point in sequence.

Why
===

I have a really nice 16TB ZFS storage pool, but unfortunately it will be a
while before I have the capital to duplicate this setup for a proper backup
system. Thus todrives was born!

Tar does support multi volume mount points, but if the need ever arises to
retrieve a single file... It can literally take days to get it.

Dar is a little better, but it still stores everything in an archive. I don't
need this.

How
===

As one drive is filled, todrives pauses to allow the user to mount another
drive to the same mount point and then continues when the enter key is pressed.

The files are copied to the dest mount point and given a UUID as the file name.
A separate log file is made that maps the UUID to file metadata such as name,
owner, group, mod time, and original path. Lose this log file, and your files
are as good as gone.

No compression is done at all. todrives goes as fast as the hardware allows!

Warning
=======

* This is currently alpha software. Only use it if you know what you are doing.

* There is no way to retrieve files yet.

* Recommended usage is to encrypt the drive before mounting!
