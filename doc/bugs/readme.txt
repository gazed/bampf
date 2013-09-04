Bugs are processed by doc/publish to generate the current bug list into chassis/target/bugs.html.
Each file is expected to be in pandoc/markdown format with the date and bug summary as the title.

Usage Notes:
  o There is one bug per file to reduce merge conflicts.
  o Bugs are to be deleted once they have been tested as fixed.

The bug system is currently home grown because:
  o Bug lists should be small (more design/refactor/unit-test if this is not the case).
  o The bug list stays with the code and is automatically versioned.
  o There is no need to install and maintain a separate bug tracking server.
  o The existing technology base can be reused: python, pandoc, golang web server.
