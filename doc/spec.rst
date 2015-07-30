.. -*- coding: utf-8 -*-
.. sectnum::

=================
gds Specification
=================
:Created: Sun Jul 12 14:29 2015
:Modified: Wed Jul 29 14:26 2015

.. -----
.. Inbox
.. -----

.. * Recommended usage is to encrypt the device before mounting!
.. * First run should setup config files.

------------
Introduction
------------

Building large home storage pools is expensive, and backing up this data is
critical. Duplicating a large storage system for backups could be financially
impractical for some. ``gds`` makes it simple to backup files to multiple
dissimilar devices in a cost-effective manner.

.. contents::

---------
Rationale
---------

The simplest and cheapest solution is backing up to externaly attached storage.
But the process is error prone and not efficient. Most of the time, tar is an
excellent choice for backing up data. But recovering a single file from a
multi-volume and multi-terabyte tar archive can take days.

Dar (Disk ARchiver) tool is a little better, but it still stores everything in
an binary archive format, and the UI is very complex with hundreds of options
to contend with.

-----
Goals
-----

1. **Simple**

   Provide a simple documentated interface for easily syncing files to multiple
   dissimilar storage devices.

#. **Do one thing**

   Sync files to multiple dissimilar mounted storage devices (in parrellel if
   the user desires).

#. **Correctness**

   Make sure the file on the destination matches the source.

#. **Easy recovery**

   The files stored in the backup device can be restored using standard file
   copy mechanisms.

-------
Support
-------

* Development is done on Arch Linux.

* gds primarily supports Linux. Mac OSX and Windows support may be added
  if a developer wants to step up and support it at a later date.

* gds users should understand operating system mount points and filesystem
  technologies.

* gds does not modify, compress, encrypt, or encode files at any point and
  never will. These are better left to a filesystem technology.

* gds is not network aware nor will it ever be.

* gds can cause data loss if not used properly. The gds contributors
  cannot be responsible for data loss or data recovery.

--------------
Implementation
--------------

This section details how the implemenation will meet the defined goals.

Primary use case
++++++++++++++++

The user has 20TB of data that they have collected over the years and is
priceless. They know this data is valuable, and the hardware they are storing
it on is comodity, but the costs for professional solutions are just too high.
This cost limitation makes backups a chore, and puts the data at severe risk.

gds allows the user to backup all of their data to anything that can be
mounted to a mount point in their OS and verify the data to be correct. The
user is excited, so they scrounge up devices that have been sitting in a shoe
box on the top shelf in the closet. The user needs a way to attach the device to
the storage device, so they purchase an external usb dock at local electronics
warehouse.

Illustrative use cases
++++++++++++++++++++++

Typical use cases of the gds program.

First run
~~~~~~~~~

When the program is first used, and a configuration file does not exist in the
expected default config locations, then the program will ask the user for
source directory, output directory, and the current device name (mounted at the
given mount point). Once these options are determined, the program will begin
syncing the data.

Multi-device operation
~~~~~~~~~~~~~~~~~~~~~~

When the first device is filled, the user will be asked to mount the second
device. If the the current run is the first run, then the user will be prompted
for a device name. Once this is done the operation will continue until all the
data is copied. When the program is finished, statistics are dumped to the
console and a catalog file is saved to the configuration directory in json
format. The catalog file name contains the date.

Successive runs
~~~~~~~~~~~~~~~

It's been a few weeks and the user wants to update the backup, so they initiate
gds. gds checks the mounted device and it is not similar to the
device listed in the configuration file (based on saved UUID). gds prompts
the user to mount a correct device, or force overwrite of the currently mounted
device. The user wants to replace the device, so they select "Force overwrite".
gds updates the configuration for the new device and begins syncing the
data.

Once the new device is filled, gds prompts the user to mount the second
device. The new device was larger, so some of the files that were on the second
device are now on the first device, so gds removes those files, but there
still are some files that need to be updated on the second device. gds
uses the rsync algorithm to sync the changed files efficiently. The process
continues until done.

.. Thought experiment: What happens when large files are added to a directory
   that is saved on one device, but that device is full?

Missing catalog file
--------------------

If the catalog is missing or corrupt, the user would be prompted to restore a
copy of the catalog and given options to retry, or continue. If the catalog is
restored, gds will continue normally. If the catalog is not restored, the
user will be notified again that data loss may occurr on the destination
devices. If the user continues, then gds will do a normal sync to the
devices updating changed files and removing files that are missing at the
source directory.

Large files
~~~~~~~~~~~

If the files for backup are too large for one device, then the file will be
split across devices. This metadata will be stored in the catalog. If the
``--no-split`` argument is used then the program will exit.

.. TODO: How to handle split files with the rsync algorithm?
.. TODO: How to handle split files and changed device lists. I.e., user changes
         a device to a larger or smaller device in the middle of the run.

File recovery
~~~~~~~~~~~~~

There are multiple ways a file can be recovered from a gds backup.

Using gds for recovery
----------------------

The user searches the catalog for the file they are looking for using the
``--search=<regex>`` command argument. Once found, they use
``--recover=<regex>`` to recover the files they desire. gds will prompt
the user to mount the device containing the file. After the user has indicated
they would like to continue, gds will sync the globbed files to the
original location saved in the catalog, or to the specified path using the
``--output=<path>`` command argument.

Using standard tools for recovery
---------------------------------

TODO

Parallel sync
-------------

If the user has specified multple destination mount points in ``config.yml``,
then gds will sync to those number of mount points concurrently.

Third-party libraries
+++++++++++++++++++++

* Building

  https://github.com/constabulary/gb

  Per project build tool. Gives us more flexibility in the future around how
  the gds project is organized.

* cli support

  https://github.com/codegangsta/cli

  Simplifies command-line argument handling and application structure.

* logging

  - logrus

  - go-spew

Configuration
+++++++++++++

gds checks the following paths for configuration files (in order)::

    "--config" argument passed to gds
    $XDG_CONFIG_DIR/gds/config.yml
    $HOME/.gds/config.yml
    /etc/gds/config.yml

config.yml
~~~~~~~~~~

- Multiple backup source directories.

  If the path ends with a "/" then only the contents of the path are saved to
  the destination. If a path does not end with a "/", then the path and the
  contents are saved to the destination.

- Multiple destination directories.

  In this case gds will backup in parallel.

- A list of backup devices.

  This list is auto-generated when gds is first run and the user does not
  provide a list.

  - Device name provided by the user
  - Mounted partition size
  - Mounted partition UUID

Command arguments
~~~~~~~~~~~~~~~~~

Written in docopt_ syntax.

::

    -h          --help              Show help.
    -v          --version           Show version number.
    -c=<file>   --config=<file>     Configuration file to use.
    -s=<regex>  --search=<regex>    Search the catalog for files.
    -r=<regex>  --recover=<regex>   Recover files.
    -o=<path>   --output=<path>     Recover files to path.
    -V          --verify-dest       Check hash of destination file once copied.
    -l          --list-splits       Show multi-device files.
    -n          --no-split          Do not split files across devices.

Catalog file
++++++++++++

After a successful run, gds dumps a catalog file to the configuration
directory named ``2015-07-12T21:11-catalog.json``. This file is a the file list
object from within the program encoded into json.

The catalog is needed for faster recovery of files and in the case of files
being split across devices.

The catalog should be backed up and protected just-in-case.

File sync
+++++++++

- Uses rsync algorithm

- Split files for large files (unless ``--no-split`` is used).

Recovery
++++++++

Files are synced directly to the device without modification unless the file
was split across devices because it was too big.

.. _docopt: http://docopt.org
