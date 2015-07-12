=====================================
todrives -- Easy multi-device backups
=====================================

todrives syncs files from one large storage pool, to multiple dissimilar
drives— one drive at a time (for now).

.. warning:: This is alpha software. Use only if you know what you are doing.


------
How-to
------

1. Get source

   .. code:: console

      git clone https://github.com/demizer/todrives.git --recursive

#. Build and use

   On the first run, todrives will use ``$EDITOR`` (default is vim) to edit
   config files.

   .. code:: console

      go install && todrives
