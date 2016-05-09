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

* Install the [vu](https://github.com/gazed/vu) engine first using ``go get github.com/gazed/vu``.
* Download bampf into the ``src`` directory of a Go workspace which is
  any directory in the ``$GOPATH``. Using just ``go get github.com/gazed/bampf``
  places bampf in ``$GOPATH/src/github.com/gazed/bampf`` and works for producing
  developer builds, but not production builds.
* Create developer builds using ``go build`` from the ``bampf`` directory.
  Run the game ``./bampf``.
* Create shippable product builds using ``build.py`` from ``bampf/admin``.
  All build output is located in the ``bampf/admin/target`` directory. Eg:
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

* Same dependency limitations as the [vu](https://github.com/gazed/vu) engine.
* Production builds use zip. On Windows there is a WIN 64-bit zip available at
  willus.com/archive/zip64. Put zip.exe in PATH.
