#! /usr/bin/python
# Copyright (c) 2013-2015 Galvanized Logic Inc.
# Use is governed by a BSD-style license found in the LICENSE file.

"""
The build and distribution script for the Bampf project.

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

def clean():
    # Remove all generated files.
    generatedOutput = ['pkg', 'bin', 'target']
    print 'Removing generated output:'
    for gdir in generatedOutput:
        if os.path.exists(gdir):
            print '    ' + gdir
            shutil.rmtree(gdir)

def buildSrc():
    # Builds executable.
    if sys.platform.startswith('darwin'):
        buildOSX()
    elif sys.platform.startswith('linux'):
        buildLinux()
    elif sys.platform.startswith('win'):
        buildWindows()
    else:
        print 'No build for ' + sys.platform

def buildBinary(flags):
    print 'Building executable'
    subprocess.call(shlex.split('go fmt bampf'))
    try:
        version = subprocess.check_output(shlex.split('git describe')).strip()
    except subprocess.CalledProcessError:
        version = 'v0.0'
    command = 'go build -ldflags "-X main.version '+version+' '+flags+'" -o target/bampf.raw bampf'
    out, err = subprocess.Popen(command, universal_newlines=True, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE).communicate()
    print('built binary with command: ' + command)

def zipResources():
    # zip the resources and include them with the binary.
    # chdir to get resource file zip proper names.
    cwd = os.getcwd()
    os.chdir('src/bampf')
    subprocess.call(['zip', 'resources.zip']+glob.glob('models/*')+glob.glob('source/*')+glob.glob('images/*')+glob.glob('audio/*'))
    os.chdir(cwd)
    shutil.move('src/bampf/resources.zip', 'target/resources.zip')

def buildOSX():
    print 'Building the osx application bundle.'
    buildBinary('-linkmode=external')
    subprocess.call(shlex.split('mv target/bampf.raw target/bampf'))
    subprocess.call(shlex.split('chmod +x target/bampf'))
    zipResources()

    # create the OSX application bundle directory structure.
    base = 'target/Bampf.app'
    if os.path.exists(base):
        shutil.rmtree(base)
    base = 'target/Bampf.app/Contents'
    os.makedirs(base + '/MacOS')
    os.makedirs(base + '/Resources')
    os.makedirs(base + '/Frameworks')

    # create the osx bundle by putting everything in the proper directories.
    subprocess.call(shlex.split('cp src/Info.plist target/Bampf.app/Contents/'))
    subprocess.call(shlex.split('cp target/bampf target/Bampf.app/Contents/MacOS/Bampf'))
    subprocess.call(shlex.split('cp target/resources.zip target/Bampf.app/Contents/Resources/'))
    subprocess.call(shlex.split('cp src/bampf.icns target/Bampf.app/Contents/Resources/Bampf.icns'))

    # change the directory mode to be an application package and make a copy for app store signing.
    shutil.copymode('/Applications/Contacts.app', base)
    if os.path.exists('target/app'):
        shutil.rmtree('target/app')
    os.makedirs('target/app')
    subprocess.call(shlex.split('cp -r target/Bampf.app target/app/Bampf.app'))

def buildWindows():
    print 'Building windows'

    # create the icon resource to include with the binary.
    cwd = os.getcwd()
    os.chdir('src')
    subprocess.call(shlex.split('windres bampf.rc -O coff -o bampf/resources.syso'))
    os.chdir(cwd)

    # build the raw binary and cleanup the generated icon (windows resource) file.
    buildBinary('-H windowsgui')
    os.remove('src/bampf/resources.syso')

    # combine the exe and the resources. Need to redirect output for cat to work.
    zipResources()
    with open('target/bampf', "w") as outfile:
        subprocess.call(['cat', 'target/bampf.raw', 'target/resources.zip'], stdout=outfile)
    subprocess.call(shlex.split('zip -A target/bampf'))
    subprocess.call(shlex.split('mv target/bampf target/Bampf.exe'))

def buildLinux():
    print 'TODO Building linux'

#------------------------------------------------------------------------------
# Main program.

def usage():
    print 'Usage: build [clean] [src]'

if __name__ == "__main__":
    options = {'clean'  : clean,
               'src'    : buildSrc}
    somethingBuilt = False
    for arg in sys.argv:
        if arg in options:
            print 'Performing build ' + arg
            options[arg]()
            somethingBuilt = True
    if not somethingBuilt:
        usage()
