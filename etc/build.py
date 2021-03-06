#! /usr/bin/python
# Copyright (c) 2013-2016 Galvanized Logic Inc.
# Use is governed by a BSD-style license found in the LICENSE file.

"""
The build and distribution script for the Bampf project.
Expected to be run from this directory.
All build output placed in a local 'target' directory

This script detects and builds for the platform that it is on.  All the build
knowledge for any computer architecture is contained in this script.
Note that build commands are specified in such a way that they can be easily
copied and tested in a shell.

This script is expected to be called by:
   1) a continuous integration script from a dedicated build server, or,
   2) a local developer testing the build.
"""

import sys          # detect which arch for the build
import os           # for directory manipulation
import signal       # for process signal definitions.
import shutil       # for directory and file manipulation
import shlex        # run and control shell commands
import subprocess   # for calling shell commands
import glob         # for unix pattern matching

def cleanProject():
    # Remove all generated files.
    generatedOutput = ['target']
    print 'Removing generated output:'
    for gdir in generatedOutput:
        if os.path.exists(gdir):
            print '    ' + gdir
            shutil.rmtree(gdir)

def lintProject():
    # expects golint executable in $PATH
    run('golint bampf')

def buildProject():
    # Builds executable.
    if sys.platform.startswith('darwin'):
        buildOSX()
    elif sys.platform.startswith('win'):
        buildWindows()
    else:
        print 'No build for ' + sys.platform

def buildBinary(flags):
    print 'Building executable'
    run('go fmt bampf')
    try:
        version = subprocess.check_output(shlex.split('git describe')).strip()
    except subprocess.CalledProcessError:
        version = 'v0.0'
    command = 'go build -ldflags "-s -X main.version='+version+' '+flags+'" -o target/bampf.raw bampf'
    out, err = subprocess.Popen(command, universal_newlines=True, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE).communicate()
    print('built binary with command: ' + command)

def zipAssets():
    # zip the resources and include them with the binary.
    # chdir to get resource file zip proper names.
    cwd = os.getcwd()
    os.chdir('..')
    subprocess.call(['zip', 'assets.zip']+glob.glob('models/*')+glob.glob('source/*')+
                    glob.glob('images/*')+glob.glob('audio/*'))
    os.chdir(cwd)
    shutil.move('../assets.zip', 'target/assets.zip')

def buildOSX():
    print 'Building the osx application bundle.'
    buildBinary('-linkmode=external')
    run('mv target/bampf.raw target/bampf')
    run('chmod +x target/bampf')
    zipAssets()

    # create the OSX application bundle directory structure.
    base = 'target/Bampf.app'
    if os.path.exists(base):
        shutil.rmtree(base)
    base = 'target/Bampf.app/Contents'
    os.makedirs(base + '/MacOS')
    os.makedirs(base + '/Resources')

    # create the osx bundle by putting everything in the proper directories.
    run('cp Info.plist target/Bampf.app/Contents/')
    run('cp target/bampf target/Bampf.app/Contents/MacOS/Bampf')
    run('cp target/assets.zip target/Bampf.app/Contents/Resources/')
    run('cp bampf.icns target/Bampf.app/Contents/Resources/Bampf.icns')

    # Create a signed copy for self distribution.
    if os.path.exists('target/dist'):
        shutil.rmtree('target/dist')
    os.makedirs('target/dist')
    run('cp -r target/Bampf.app target/dist/Bampf.app')
    pkgOSX('target/dist', '"Developer ID Application: XXX"', '"Developer ID Installer: Paul Ruest"')

    # Create a signed copy for app store submission.
    if os.path.exists('target/app'):
        shutil.rmtree('target/app')
    os.makedirs('target/app')
    run('cp -r target/Bampf.app target/app/Bampf.app')
    pkgOSX('target/app', '"3rd Party Mac Developer Application: Galvanized Logic Inc."',
           '"3rd Party Mac Developer Installer: Galvanized Logic Inc."')

def pkgOSX(outdir, akey, ikey):
    run('codesign --force --entitlements Entitlements.plist --sign '+
        akey+' --timestamp=none '+outdir+'/Bampf.app')
    run('productbuild --version 1.0 --sign '+ikey+' --component '+
        outdir+'/Bampf.app /Applications '+outdir+'/Bampf.pkg')

def buildWindows():
    print 'Building windows'

    # create the icon resource to include with the binary.
    run('windres bampf.rc -O coff -o ../resources.syso')

    # build the raw binary and cleanup the generated icon (windows resource) file.
    buildBinary('-H windowsgui')
    os.remove('../resources.syso')

    # combine the exe and the resources. Need to redirect output for cat to work.
    zipAssets()
    with open('target/bampf', "w") as outfile:
        subprocess.call(['cat', 'target/bampf.raw', 'target/assets.zip'], stdout=outfile)
    run('zip -A target/bampf')
    run('mv target/bampf target/Bampf.exe')

def run(command):
    # execute command in the shell.
    subprocess.call(shlex.split(command))

#------------------------------------------------------------------------------
# Main program.

def usage():
    print 'Usage: build [clean] [lint] [src]'

if __name__ == "__main__":
    options = {'clean'  : cleanProject,
               'lint'   : lintProject,
               'src'    : buildProject}
    somethingBuilt = False
    for arg in sys.argv:
        if arg in options:
            print 'Performing build ' + arg
            options[arg]()
            somethingBuilt = True
    if not somethingBuilt:
        usage()
