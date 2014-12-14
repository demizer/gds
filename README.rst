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

How
===

As one drive is filled, todrives pauses to allow the user to mount another
drive to the same mount point and then continues when the enter key is pressed.

The files are copied to the dest mount point and given a UUID as the file name.
A separate log file is made that maps the UUID to file metadata such as name,
owner, group, mod time, and original path.

No compression is done at all. todrives goes as fast as the hardware allows!

Warning
=======

* This is currently alpha software. Do not use it!

* There is no way to easily retrieve files yet.

* Recommended usage is to encrypt the drive before mounting!
