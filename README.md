# ARI application

ARI is an asynchronous API that allows developers to build communications applications by exposing the raw primitive objects in Asterisk - channels, bridges, endpoints, media, etc. - through an intuitive REST interface. The state of the objects being controlled by the user are conveyed via JSON events over a WebSocket. <br/> 
More information about ARI could be find here https://wiki.asterisk.org/wiki/pages/viewpage.action?pageId=29395573 <br/> 
This application was built using library, which could be find here https://github.com/CyCoreSystems/ari <br/> 

## Features

This is the CLI application that waits for the user’s input and based on that input, it will execute some actions.
Once an action is registered, the terminal should be free for entering a new command. 
The commands that are implemented are “dial”, “list”, and “join”. <br/> 
Dial initiates a call between X endpoints. <br/> 
List prints all ongoing calls. <br/> 
Join allows joining an ongoing call. <br/> 

