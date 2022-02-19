<p align="center">
<h1 align="center">Warp! - Your lazy command line ssh helper</h1>

![](https://img.shields.io/badge/license-MIT-green.svg?style=flat)
<a href="https://www.buymeacoffee.com/addy3494" target="_blank"><img src="https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png" alt="Buy Me A Coffee" style="height: 30px !important;width: 130px !important;box-shadow: 0px 3px 2px 0px rgba(190, 190, 190, 0.5) !important;-webkit-box-shadow: 0px 3px 2px 0px rgba(190, 190, 190, 0.5) !important;" ></a>

*Manage your infrastructure, Never loose/forget passwords again.*

***
## Key Features:
- One time setup, Initialize a local database to store all your infrastructure infomation
- Supports all Linux, Unix, MacOS systems as authentication is via ssh

***
## Requirements
- python3
***
## Configure
* VIA GIT
  ```
    git clone https://github.com/psadi/warp && cd warp
    pip install -r requirements.txt
    python3 /path/to/warp.py
  ```
* VIA PIP
  ```
    pip install warp-py
    python3 -m warp (or) warp
  ```
* MANUAL INSTALL
  ```
    git clone https://github.com/psadi/warp && cd warp
    python3 setup.py install
    python3 -m warp
  ```
***
## Disclaimer

### This repository,
* Is Created to tackle my personal use-case.
* Is not production ready/safe.
* Is just a wrapper *(quality-of-life improvements)* of the existing details which you already have.
* Assumes you already have available connection for key-based auth and will not create/establish any.
* Will not take any responsibility of damage-dealt/passwords-leaks etc. It is assumed you are using this package in a controlled environment.

***
## Contributing
Bug reports and pull requests are welcome on GitHub at [warp]( https://github.com/psadi/warp ) repository.

This project is intended to be a safe, welcoming space for collaboration and contributors are expected to adhere to the
[Contributor Covenant](http://contributor-covenant.org) code of conduct.

  1. Fork it ( https://github.com/psadi/warp )
  1. Create your feature branch (`git checkout -b my-new-feature`)
  1. Commit your changes (`git commit -am 'Add some feature'`)
  1. Push to the branch (`git push origin my-new-feature`)
  1. Create a new Pull Request

***
## Author
* **psadi** - *Owner* - [psadi](https://github.com/psadi)
***

The project is available as open source under the terms of the [MIT License](LICENSE)
