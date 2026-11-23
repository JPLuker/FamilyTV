# Accessibility requirements (draft)

Accessibility is configured by individual need, not by a required diagnosis. Profiles can combine captions, audio description, screen readers, focus visibility, scaling, reduced motion, simplified navigation and alternative inputs.

## Release gates
- Parent web UI targets WCAG 2.2 AA, validated with assistive technologies.
- TV app supports D-pad/OK/Back and stable focus without touch.
- Blind/low-vision users get descriptive controls and audio-described media where available.
- Deaf/hard-of-hearing users get captions and visual alternatives for audio status.
- Switch users are not forced into precise gestures or timed interactions.
- Session-ending cues have non-audio equivalents; music is optional.
- Library Health distinguishes verified, missing and unknown accessibility metadata.
- Accessibility hard filters must never silently relax.

No claim of universal accessibility before testing with disabled users and caregivers.

## Implemented viewer settings

Each profile can independently require verified captions and a parent-verified described default soundtrack. Picture navigation uses actual frames from approved videos. Text navigation omits frame requests. Profiles can increase text size, enable high contrast and request reduced motion. The end screen shows a fact and optional music. For Compose, supply a household-owned session-ending.ogg under the media root (up to 2 MiB) and enable ending music for the profile. The player displays audio controls when a track is available.
