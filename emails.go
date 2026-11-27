package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// mailLink is a labelled URL in an email, e.g. "Join link".
type mailLink struct{ Label, URL string }

// mailView is the content of an email to one Participant. The plain-text and
// the HTML version (web/templates/email.html) are rendered from it.
type mailView struct {
	T        Messages
	Greeting string
	Intro    string     // paragraphs separated by a blank line
	Links    []mailLink // zero or more, each shown as a button
	Wish     string     // optional, the recipient's Wish
}

// Paras returns Intro split into paragraphs.
func (v mailView) Paras() []string { return strings.Split(v.Intro, "\n\n") }

// send emails the view to the Participant, greeting them by name.
func (a *app) send(p Participant, subject string, v mailView) error {
	v.Greeting = fmt.Sprintf(v.T["mail_greeting"], p.Name)
	var text strings.Builder
	fmt.Fprintf(&text, "%s\n\n%s\n", v.Greeting, v.Intro)
	for _, l := range v.Links {
		fmt.Fprintf(&text, "\n%s:\n%s\n", l.Label, l.URL)
	}
	if v.Wish != "" {
		fmt.Fprintf(&text, "\n%s:\n%s\n", v.T["draw_wish"], v.Wish)
	}
	var html bytes.Buffer
	if err := templates.ExecuteTemplate(&html, "email.html", v); err != nil {
		return err
	}
	return a.mailer.Send(Mail{To: p.Email, Subject: subject, Text: text.String(), HTML: html.String()})
}

// sendLinks emails a joined Participant their links: the Organizer all three
// (Session, Join and Personal link) in the Session's language, anyone else
// their Personal link in their own language.
func (a *app) sendLinks(s Session, p Participant) error {
	if !p.Organizer {
		return a.sendJoinConfirmation(p)
	}
	t := catalog[s.Lang]
	return a.send(p, t["organizer_subject"], mailView{T: t, Intro: t["organizer_intro"], Links: []mailLink{
		{t["session_link"], a.baseURL + "/session/" + s.Hash},
		{t["join_link"], a.baseURL + "/join/" + s.JoinHash},
		{t["personal_link"], a.baseURL + "/" + p.Hash},
	}})
}

// sendJoinConfirmation emails the joiner their Personal link in their language.
func (a *app) sendJoinConfirmation(p Participant) error {
	t := catalog[p.Lang]
	return a.send(p, t["join_subject"], mailView{T: t, Intro: t["join_intro"], Links: []mailLink{{t["personal_link"], a.baseURL + "/" + p.Hash}}})
}

// notifyOrganizer emails the Organizer, in their language, who has joined.
func (a *app) notifyOrganizer(sess Session, joiner Participant) error {
	t := catalog[sess.Lang]
	return a.send(*sess.organizer(), fmt.Sprintf(t["joined_subject"], joiner.Name), mailView{T: t, Intro: fmt.Sprintf(t["joined_intro"], joiner.Name)})
}

// sendDrawEmails tells every Participant of the drawn Session, in the
// language of their last Submission, whom they give to and that person's
// Wish; nil (no Draw happened) sends nothing. A failed email does not stop
// the others: the Draw is already saved and can't be repeated.
func (a *app) sendDrawEmails(s *Session) error {
	if s == nil {
		return nil
	}
	var errs []error
	for _, p := range s.Participants {
		t := catalog[p.Lang]
		view := mailView{T: t, Intro: fmt.Sprintf(t["draw_intro"], p.Recipient), Wish: s.participantNamed(p.Recipient).Wish}
		errs = append(errs, a.send(p, t["draw_subject"], view))
	}
	return errors.Join(errs...)
}

// notifyNamesAdded tells a joined Participant, in their language (the
// Session's if they have made no Submission yet), which names were added and
// links their Personal link, where they can still change their Exclusions.
func (a *app) notifyNamesAdded(s Session, p Participant, added []string) error {
	if p.Lang == "" {
		p.Lang = s.Lang
	}
	t := catalog[p.Lang]
	return a.send(p, t["names_added_subject"], mailView{T: t, Intro: fmt.Sprintf(t["names_added_intro"], strings.Join(added, ", ")),
		Links: []mailLink{{t["personal_link"], a.baseURL + "/" + p.Hash}}})
}

// notifyExclusionsReset tells a Participant, in their language, that removing
// names made them lose Exclusions and that they must confirm their Submission
// again on their Personal link.
func (a *app) notifyExclusionsReset(s Session, p Participant, removed []string) error {
	if p.Lang == "" {
		p.Lang = s.Lang
	}
	t := catalog[p.Lang]
	return a.send(p, t["exclusions_reset_subject"], mailView{T: t, Intro: fmt.Sprintf(t["exclusions_reset_intro"], strings.Join(removed, ", ")),
		Links: []mailLink{{t["personal_link"], a.baseURL + "/" + p.Hash}}})
}
