bampf
=====

Bampf is a simple 3D arcade style game. Collect energy cores in order to finish 
a level. Teleport (bampf) to safety or use cloaking abilities to avoid sentinels.

Bampf was created primarily to test the ``vu`` engine.  Its levels are used to 
benchmark the engine by substantially increasing the number of triangles rendered each level.  
As such the game isn't really meant to be winnable given the large number of AI's in the 
later levels. 

Build
-----

* Download and build the ``vu`` engine first. 
* Ensure GOPATH contains the ``bampf`` directory.
* Build from the ``bampf`` directory using ``./build src`` or ``python build src``. 
  All build output is located in the ``target`` directory. 
* Developer builds can be created using ``go build`` in ``bampf/src/bampf`` 
  and running ``./bampf``.

**Build Dependencies**

* vu engine.
* go and standard go libraries.
* python for the build script.
* git for product version numbering.

**Runtime Dependencies**

Transitive dependencies from the ``vu`` engine.

* OpenGL version 3.2 or later.
* OpenAL 64-bit version 2.1.

Limitations
-----------

* Windows is limited by the availability of OpenGL and OpenAL. Generally 
  downloading the graphics card maufacturer's drivers fixes OpenGL issues.  
  However laptops with Intel graphics don't always have OpenGL drivers. 
* 64-bit OpenAL may be difficult to locate for Windows machines. See 
  http://connect.creativelabs.com/openal/Downloads/oalinst.zip if/when their website is up.
* Building on Windows used golang with gcc from mingw64-bit. 
  Building with Cygwin may have special needs. 
