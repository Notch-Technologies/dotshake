#!/bin/sh
uninstall() {
  if type "systemctl" > /dev/null 2>&1; then
    printf "\e[31mstopping the dotshake...\e[m\n"
    sudo systemctl stop dotshaker.service || true
    if [ -e /lib/systemd/system/dotshake.service  ]; then
      rm -f /lib/systemd/system/dotshake.service 
      systemctl daemon-reload || true
      printf "\e[31mremoved dotshake.service and reloaded daemon.\e[m\n"
    fi
  
  else
    printf "\e[31muninstalling the dotshake...\e[m\n"
    /usr/bin/dotshaker daemon uninstall || true
  
    if type "systemctl" > /dev/null 2>&1; then
       printf "\e[31mrunning daemon the reload\e[m\n"
       systemctl daemon-reload || true
    fi
  
  fi
}

uninstall
