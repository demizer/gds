.. -*- coding: utf-8 -*-
.. sectnum::

======================
todrives Specification
======================
:Created: Sun Jul 12 14:29 2015
:Modified: Sun Jul 12 14:44 2015

-----
Inbox
-----

* Recommended usage is to encrypt the drive before mounting!
* First run should setup config files.

------------
Introduction
------------

Building large home storage pools is expensive for the average consumer, and
backing up this data is critical. Duplicating this system for backups would be
financially impractical. ``todrives`` makes it simple to backup files to
multiple dissimilar devices in a cost-effective manner.

.. contents::

---------
Rationale
---------

The simplest and cheapest solution is backing up to externaly attached storage.
But the process is error prone and not efficient. Most of the time, tar is an
excellent choice for backing up data. But recovering a single file from a
multi-volume multi-terabyte tar archive can take days.

Dar (Disk ARchiver) tool is a little better, but it still stores everything in
an binary archive format.

--------------
Implementation
--------------

As one drive is filled, todrives pauses to allow the user to mount another
drive to the same mount point and then continues when the enter key is pressed.

The files are copied to the dest mount point and given a UUID as the file name.
A separate log file is made that maps the UUID to file metadata such as name,
owner, group, mod time, and original path. Lose this log file, and your files
are as good as gone.

No compression is done at all. todrives goes as fast as the hardware allows!

The catalog is saved locally. It is critical that this file is backed up.
Otherwise files that are split will have to be manually merged.
