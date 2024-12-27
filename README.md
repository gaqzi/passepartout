Go templates with htmx spike
============================

The goal of this repo is for me to experiment with a slightly more advanced way of working with Go's built-in HTML templates because I'm not very familiar with them.

I have been using [donseba/go-htmx](https://github.com/donseba/go-htmx) and the component's it creates, but I'm not loving the way it comes together. I'm probably not understanding how to use them correctly, but the built-in templates look so much like jinja2 which I used to use a lot with Ansible and I really would like to see if I could get that behavior instead.

This spike is made in the context of [gaqzi/incident-reviewer](https://github.com/gaqzi/incident-reviewer).

# The rough plan

I was thinking it over and I think that basically what I want is:
1. A hierarchy of "reusable components" (kinda like partials, but bigger, more on the level of `<contributing-causes>` which in turn relies on several other partials to build itself up). I guess most of these could be standalone pages in reality but how I'm working with them is that I will want to pull them into one page __most of the time.__
2. Partials `define`d and reusable as `template <name>` which will handle things like render this particular thing
    - I will probably come up with some standard pattern for __nearly always__ explicitly passing in the data I want to them
3. layouts, probably I'll use two to start with:
    - `page` which is "surround me with all the HTML so it can be rendered standalone. Not loving the name. And then each thing that's wrapped can define:
        - `content`: where the main body goes, you define it in your "page"/component
        - `title` the name of the page
        - …whatever else comes to mind
    - `htmx` which contains nothing at all, it's just a blank template, for now, so that you can render whatever you have inline.
        - `content`: the main content just like the standard `page` layout
        - `out-of-band`: so any oob updates are put here
            - I wonder if there's a thing where if you don't call the `define`'d it won't execute? that'd be really nifty to avoid running stuff
            - There was a suggestion from Claude to pass in a variable `mode` by default so you could check in the template whether it's a full page or HTMX/partial rendering so you can decide what to do in the template, but that seems like cheating.
            - also, is OOB a decision that should be made in the template and not something where you can just ask to add on more things in the output from inside something else?
            - anyway, this isn't a real concern for now because I'm not dealing with OOBs yet
