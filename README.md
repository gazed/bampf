bampf
=====

Bampf is a simple 3D arcade style game. Collect energy cores in order to finish 
a level. Teleport (bampf) to safety or use cloaking abilities to avoid sentinels.

Bampf was created primarily to test the ``vu`` engine.  Its levels are used to 
benchmark the engine by substantially upping the number-of-triangles-rendered each level.  
As such the game isn't really meant to be winnable given the silly number of AI's in the 
later levels. 

Build
-----

Download and build the ``vu`` engine first.  Then build bampf using ``./build src`` or ``python build src``.
All build output is located in the ``target`` directory.

**Build Dependencies**

Dependencies are kept to a minimum where possible. In general everything is text based
(better for source control) and can be developed in a command line environment (not saying
IDE's are bad, just that there are no IDE dependencies or project files).

* vu engine.
* go and standard go libraries. No external go packages outside of ``vu`` are used.
* python for the source and documentation build scripts.
* pandoc for building documentation.
* git for source control and product version numbering.
* ``osx`` build/development needs ``DYLD_LIBRARY_PATH`` set to find the ``vu/device/libvudev.1.dylib``.
  e.g. ``export DYLD_LIBRARY_PATH=$HOME/projects/vu/device/libvudev.1.dylib``.
  This dependency may go away with golang 1.2.

Ignore python and pandoc if you want build by hand using ``go install``.  Check the 
build script for the order.

**Runtime Dependencies**

Transitive dependencies from the ``vu`` engine.

* OpenGL version 3.2 or later.
* OpenAL 64-bit version 2.1.

Limitations
-----------

* Windows is limited by the availability of OpenGL and OpenAL.  Generally 
  downloading the graphics card maufacturer's drivers fixes OpenGL issues.  
  However laptops with Intel graphics don't always have OpenGL drivers. 
* 64-bit OpenAL may be difficult to locate for Windows machines 
  (http://connect.creativelabs.com/openal/Downloads/oalinst.zip if/when their website is up).
* Windows building has only been done using golang with gcc from mingw64-bit. 
  Building with Cygwin may have special needs. 
