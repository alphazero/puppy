# puppy

---

![Smaller icon](http://photos1.blogger.com/x/blogger/607/785/1600/106809/silvertone.jpg "bite me")

---

##about

`puppy` is a toy console based log stream monitor, written per a **non-paying non-client's spec & to waste my very (very) precious time**. It is my very first piece of software literally written in anger (so now I know what that means! ;). 

You point it to a `W3C Common Log Format` file and it will tail it, with periodic snapshot of various access metrics, and interminent checks of traffic volume. 

##Supported Platforms
`puppy` requires `tail`, and `stty` to operate. Not sure? Run it and it will let you know if it will play with you.

##usage

    puppy -f <path> [optioal flags]

On startup, `puppy` defaults to ()snapshot) stats view. You can switch views at anytime (per commands below). All views (except the debug view) provide a uniform header and footer. 

Alert status is shown in the footer. To see the historic list, switch to the alerts view. (Paging/scrolling is possible todo and you may volunteer patches if you care! :) 

###interaction
Your `puppy` recognizes the following cues. (Note that it doesn't use `raw mode` so its a bit cluncky -- press the single command char and then `enter`.)

    switch to stats view:       s | S 
    switch to alerts view:      a | A
    switch to log view:         l | L 
    quit:                       q | Q
    

###known bugs
`puppy` makes best effort attempts to shutdown cleanly, specially given the fact that it modifies disables `TTY` `echo` and launches `tail`.  This all works fine except if you kill -9 `puppy`, in which case (for some as of yet unresolved issue) the `tail` process hangs around, and you will need to issue 'ps` to fish out the tail process id and manually terminate it. 

###extensibility
Care has been take to allow you, my dear fellow geek, to jump in and hack to extend `puppy`. Duly commented.
