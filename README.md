<!-- Copyright © 2013-2016 Galvanized Logic Inc.                       -->
<!-- Use is governed by a BSD-style license found in the LICENSE file. -->

#Bampf

Bampf is a simple 3D arcade style game. Collect energy cores in order to finish
a level. Teleport (bampf) to safety or use cloaking abilities to avoid sentinels.

Bampf was created primarily to test the [vu](https://github.com/gazed/vu) 3D engine.
Its levels are used to benchmark the engine by substantially increasing the number
of triangles rendered each level. As such the game isn't really meant to be winnable
given the large number of AI's in the later levels.

Build
-----

* Install the [vu](https://github.com/gazed/vu) engine first.
* Create developer builds using ``go build`` in ``bampf/src/bampf``.
  Run the game ``./bampf``.
* Create product builds using ``build.py`` in ``bampf``. All build output
  is located in the ``target`` directory. Eg:
    * OS X:

        ```bash
        ./build.py src
        open target/Bampf.app
        ```
    * Windows:

        ```bash
        python build.py src
        target/bampf.exe
        ```

**Developer Build Dependencies**

* go1.6
* vu engine.

**Production Build Dependencies**

* go1.6
* vu engine.
* python for the build script.
* git for product version numbering.
* zip for appending resources to the binary.

**Runtime Dependencies**

Transitive dependencies from the ``vu`` engine.

* OpenGL version 3.3 or later.
* OpenAL 64-bit version 2.1.

Limitations
-----------

* Same dependency limitations as the vu engine.
* Production builds use zip. On Windows there is a WIN 64-bit zip available at
  willus.com/archive/zip64. Put zip.exe in PATH.
