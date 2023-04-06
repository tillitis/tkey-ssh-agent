
# Using Wix on Windows to build the msi package

The following might be missing something, please revise.

- You probably want to be on a tagged version of the apps repo.

- Run `make windows` in the apps repo. I guess this currently only
  works on Linux.

- Place the built `tkey-ssh-agent.exe` and `tkey-ssh-agent-tray.exe`
  in this directory.

- Install Wix 3:
  - Install the required .NET 3.5:
    - Go to `Control Panel » Programs and Features » Turn Windows
      features on or off`
    - Tick `.NET Framework 3.5 …` (and the sub headings), and let it
      install.
  - Download and install:
    https://github.com/wixtoolset/wix3/releases/tag/wix3112rtm

- Open a terminal in this directory. We have a script that builds the
  msi using Wix candle & light tools.

  The script takes the version number of the package that it should
  produce. We shall pass the typical tagged version, and a 4th digit
  (0) will be added, per windows convention.

  ```
  ./build-msi.bat 0.0.6
  ```

- You can try installing the msi with:

  ```
  msiexec /i tkey-ssh-agent-0.0.6.0.msi`
  ```

We do not enable the agent to run automatically at startup, leaving
this to the decision of the user. But we do install a shortcut in the
folder for "All Users Start Menu Programs", so it ends up on the
user's Start Menu (in `TKey SSH Agent\TKey SSH Agent`). Running this
shortcut starts the tray executable with our default arguments `--uss
-a tkey-ssh-agent`. User can copy this shortcut to their "Startup"
folder, as described in [this
article](https://support.microsoft.com/en-us/windows/add-an-app-to-run-automatically-at-startup-in-windows-10-150da165-dcd9-7230-517b-cf3c295d89dd).
Or by running the following PowerShell commands:

```PowerShell
$w = new-object -comobject wscript.shell
$prgs = $w.specialfolders('allusersprograms')
$startup = $w.specialfolders('startup')
copy "$prgs\TKey SSH Agent\TKey SSH Agent.lnk" "$startup\"
```

Also, the default configuration relies on finding a `pinentry` program
from the Gpg4win package. To install this package run:

```
winget install gpg4win
```

Since this msi will also be available as winget package, this
dependency seems fine.
