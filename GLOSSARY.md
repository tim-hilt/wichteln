# Wichteln

A self-hosted Secret Santa app: a group of people secretly draws who gives a present to whom.

## People

**Organizer**:
The **Participant** who created a **Session** and named all its **Participants**. The Organizer is always also a Participant, and never sees the other Participants' email addresses.
_Avoid_: Admin, owner, host

**Participant**:
A person taking part in a **Session**, identified by the name the **Organizer** gave them. Each Participant provides their own email address when they **Join**.
_Avoid_: User, member, player

## Session

**Session**:
One Secret Santa round, consisting of its **Participants** and, eventually, their **Assignments**.
_Avoid_: Group, game, round, event

**Session link**:
The secret URL that gives the **Organizer** access to a **Session**'s overview.
_Avoid_: Admin link

**Personal link**:
The secret URL that gives one **Participant** access to their own **Wish**, **Exclusions** and, after the **Draw**, their **Assignment**.
_Avoid_: User link, invite link

**Join link**:
The secret URL of a **Session** that the **Organizer** shares with the group, through which each **Participant** claims their name. It reveals nothing else about the **Session**.
_Avoid_: Invite link, public link

**Join**:
A **Participant** claiming their name through the **Join link**, together with their first **Submission**. Each name can be joined only once.
_Avoid_: Register, sign up

**Release**:
The **Organizer** undoing a **Join** before the **Draw**, so that the name can be joined again.
_Avoid_: Reset, unclaim

## Wishes and drawing

**Wish**:
Free text in which a **Participant** describes what they would like to receive.
_Avoid_: Wishlist, gift idea

**Exclusion**:
A one-way rule set by a **Participant** that they must not give to a specific other **Participant**.
_Avoid_: Blocklist, constraint

**Submission**:
A **Participant** confirming their **Wish** and **Exclusions**. It can be edited until the **Draw**.

**Draw**:
The one-time event that randomly creates all **Assignments** of a **Session**, respecting all **Exclusions**.
_Avoid_: Matching, shuffle

**Assignment**:
The fact that one **Participant** gives a present to exactly one other **Participant**. Everyone gives once and receives once, and nobody knows who gives to them.
_Avoid_: Match, pair, partner
