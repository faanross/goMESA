---
title: "Overview of goMESA"
weight: 1
---
        
## Overview

goMESA is an educational Command and Control (C2) framework designed to demonstrate how legitimate network protocols can be repurposed for covert communications. Specifically, it utilizes the Network Time Protocol (NTP) as its transport mechanism, embedding command and control traffic within seemingly normal time synchronization packets.

It is based on [mesa](https://github.com/d3adzo/mesa), with the following notable changes:
- Entire application is rewritten in Golang + integration of concurrency
- SQLite instead of MySQL as a simpler option
- Vue.js Web Client UI
- Docker support
- AES-256-GCM as a more secure encryption method
- Comprehensive documentation


[NEXT PAGE](../core_concepts/)


        
