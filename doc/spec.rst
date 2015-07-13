.. -*- coding: utf-8 -*-
.. sectnum::

======================
todrives Specification
======================
:Created: Sun Jul 12 14:29 2015
:Modified: Sun Jul 12 22:47 2015

.. -----
.. Inbox
.. -----

.. * Recommended usage is to encrypt the drive before mounting!
.. * First run should setup config files.

------------
Introduction
------------

Building large home storage pools is expensive, and backing up this data is
critical. Duplicating a large storage system for backups could be financially
impractical for some. ``todrives`` makes it simple to backup files to multiple
dissimilar devices in a cost-effective manner.

.. contents::

-------
Support
-------

* Development is done on Arch Linux.

* todrives supports Mac OSX and Linux. Windows may be added if a developer
  wants to step up and support it at a later date.

* todrives is built for sophisticated users. Users should understand operating
  system mount points and filesystem technologies.

* todrives assumes full destination device usage.

* todrives does not modify, compress, encrypt, or encode files at any point and
  never will. These are better left to a filesystem technology.

* todrives is not network aware nor will it ever be. Backing up to mounted nfs
  shares is perfectly fine, but todrives will fill it up eventually.

* todrives can cause data loss if not used properly. The todrives contributors
  cannot be responsible for dataloss and will not hold your hand in the case of
  traumatic data loss.

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

1. Simple: Provide a simple documentated interface for easily syncing files to
   multiple dissimilar storage devices.

#. Do one thing: Sync files to multiple dissimilar mounted storage devices (in
   parrellel if the user desires).

#. Strong emphasis on correctness: Make sure the file on the destination
   matches the source.

#. Easy recovery: The files stored in the backup device can be restored using
   standard file copy mechanisms.

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

todrives allows the user to backup all of their data to anything that can be
mounted to a mount point in their OS and verify the data to be correct. The
user is excited, so they scrounge up drives that have been sitting in a shoe
box on the top shelf in the closet. The user needs a way to attach the drive to
the storage device, so they purchase an external usb dock at local electronics
warehouse.

Illustrative use cases
++++++++++++++++++++++

Typical use cases of the todrives program.

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
device and provide a name (if being run without a config). Once this is done
the operation will continue until all the data is copied. When the program is
finished, statistics are dumped to the console and a catalog file is saved to
the configuration directory in json format. The catalog file name contains the
date.

Successive runs
~~~~~~~~~~~~~~~

TODO

Missing catalog file
--------------------

If the catalog is missing or corrupt, the user would be prompted to restore a
copy of the catalog and given options to retry, or continue. If the catalog is
restored, todrives will continue normally. If the catalog is not restored, the
user will be notified again that dataloss may occurr on the destination
devices. If the user continues, then todrives will do a normal sync to the
devices updating changed files and removing files that are missing at the
source directory.

Large files
~~~~~~~~~~~

If the files for backup are too large for one device, then the file will be
split across devices. This metadata will be stored in the catalog. If the
``--no-split`` argument is used then the program will exit.

File recovery
~~~~~~~~~~~~~

There are multiple ways a file can be recovered from a todrives backup.

Using todrives for recovery
---------------------------

The user searches the catalog for the file they are looking for. Once found,
they use ``--recover=<regex>`` to recover the files they desire. todrives will
prompt the user to mount the device containing the file. After the user has
indicated they would like to continue, todrives will sync the globbed files to
the original location saved in the catalog, or to the specified path using the
``--output=<path>`` command argument.

Using standard tools for recovery
---------------------------------

TODO

Parallel sync
-------------

If the user has specified multple destination mount points in ``config.yml``,
then todrives will sync to those number of mount points at a time
asyncronously.

Third-party libraries
+++++++++++++++++++++

* cli support

  https://github.com/codegangsta/cli

* Argument parsing

  https://github.com/docopt/docopt.go

* Output logging

  log15
  go-spew

* Debugging

  godebug

Configuration
+++++++++++++

todrives checks the following paths for configuration files (in order)::

    "--config" argument passed to todrives
    $XDG_CONFIG_DIR/todrives/config.yml
    $HOME/.todrives/config.yml
    /etc/todrives/config.yml

config.yml
~~~~~~~~~~

- Multiple backup source directories.

- Multiple destination directories.

  In this case todrives will backup in parallel.

- A list of backup devices.

  This list is auto-generated when todrives is first run and the user does not
  provide a list.

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
    -l          --list-splits       Show multi-device files.
    -n          --no-split          Do not split files across devices.

Catalog file
++++++++++++

After a successful run, todrives dumps a catalog file to the configuration
directory named ``2015-07-12T21:11-catalog.json``. This file is a the file list
object from within the program encoded into json.

The catalog is needed for faster recovery of files and in the case of files
being split across devices.

The catalog should be backed up and protected just-in-case.

Recovery
++++++++

Files are synced directly to the device without modification unless the file
was split across devices because it was too big.

.. _docopt: http://docopt.org
