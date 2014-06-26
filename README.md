bampf
=====

Bampf is a simple 3D arcade style game. Collect energy cores in order to finish
a level. Teleport (bampf) to safety or use cloaking abilities to avoid sentinels.

Bampf was created primarily to test the ``vu`` 3D engine. Its levels are used to
benchmark the engine by substantially increasing the number of triangles rendered
each level. As such the game isn't really meant to be winnable given the large
number of AI's in the later levels.

Build
-----

* Download and build the ``vu`` engine first. 
* Ensure GOPATH contains the ``bampf`` and ``vu`` directories.
* Create developer builds using ``go build`` in ``bampf/src/bampf``
  and running ``./bampf``.
* Create product builds using ``build`` in ``bampf``. All build output
  is located in the ``target`` directory. Eg:
    * OSX:
        * ``./build src``
        * ``open target/Bampf.app``
    * WIN:
        * ``python build src``
        * ``target/bampf.exe``

**Developer Build Dependencies**

* go1.3
* vu engine.

**Production Build Dependencies**

* go1.3
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

* Windows is limited by the availability of OpenGL and OpenAL. Generally
  OpenGL issues are fixed by downloading manufacturer's graphic card drivers.
  However older laptops with Intel graphics don't always have OpenGL drivers.
* 64-bit OpenAL may be difficult to locate for Windows machines.
  Try ``http://kcat.strangesoft.net/openal.html/openal-soft-1.15.1-bin.zip``
* Bampf has been built and tested on Windows using gcc from mingw64-bit.
  Mingw64 was installed to c:/mingw64.
* Put OpenAL on the gcc library path by copying
  ``openal-soft-1.15.1-bin/Win64/soft_oal.dll`` to
  ``c:/mingw64/x86_64-w64-mingw32/lib/OpenAL32.dll``
* WIN 64-bit zip available at willus.com/archive/zip64. Put zip.exe in PATH.
* Building with Cygwin has not been attempted. It may have special needs.
