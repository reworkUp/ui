// +build !windows,!darwin,!plan9

// 7 february 2014

package ui

import (
	"unsafe"
)

// #include "gtk_unix.h"
// /* because cgo seems to choke on ... */
// /* parent will be NULL if there is no parent, so this is fine */
// GtkWidget *gtkNewMsgBox(GtkWindow *parent, GtkMessageType type, GtkButtonsType buttons, char *title, char *text)
// {
// 	GtkWidget *k;
// 
// 	k = gtk_message_dialog_new(parent, GTK_DIALOG_MODAL, type, buttons, "%s", (gchar *) title);
// 	if (text != NULL)
// 		gtk_message_dialog_format_secondary_text((GtkMessageDialog *) k, "%s", (gchar *) text);
// 	return k;
// }
import "C"

func _msgBox(parent *Window, primarytext string, secondarytext string, msgtype C.GtkMessageType, buttons C.GtkButtonsType) (result chan int) {
	result = make(chan int)
	go func() {
		ret := make(chan C.gint)
		defer close(ret)
		uitask <- func() {
			var pwin *C.GtkWindow = nil
	
			// to implement parent, we need to put the GtkMessageDialog into a new window group along with parent
			// a GtkWindow can only be part of one group
			// so we use this to save the parent window group (if there is one) and store the new window group
			// after showing the message box, we restore the previous window group, so future parent == dialogWindow can work properly
			// thanks to pbor and mclasen in irc.gimp.net/#gtk+
			var prevgroup *C.GtkWindowGroup = nil
			var newgroup *C.GtkWindowGroup
	
			if parent != dialogWindow {
				pwin = togtkwindow(parent.sysData.widget)
				// we can't remove a window from the "default window group"; otherwise this throws up Gtk-CRITICAL warnings
				if C.gtk_window_has_group(pwin) != C.FALSE {
					prevgroup = C.gtk_window_get_group(pwin)
					C.gtk_window_group_remove_window(prevgroup, pwin)
				}
				newgroup = C.gtk_window_group_new()
				C.gtk_window_group_add_window(newgroup, pwin)
			}
	
			cprimarytext := C.CString(primarytext)
			defer C.free(unsafe.Pointer(cprimarytext))
			csecondarytext := (*C.char)(nil)
			if secondarytext != "" {
				csecondarytext = C.CString(secondarytext)
				defer C.free(unsafe.Pointer(csecondarytext))
			}
			box := C.gtkNewMsgBox(pwin, msgtype, buttons, cprimarytext, csecondarytext)
			response := C.gtk_dialog_run((*C.GtkDialog)(unsafe.Pointer(box)))
			C.gtk_widget_destroy(box)
	
			if parent != dialogWindow {
				C.gtk_window_group_remove_window(newgroup, pwin)
				C.g_object_unref(C.gpointer(unsafe.Pointer(newgroup)))		// free the group
				if prevgroup != nil {
					C.gtk_window_group_add_window(prevgroup, pwin)
				}		// otherwise it'll go back into the default group on its own
			}
	
			ret <- response
		}
		result <- int(<-ret)
	}()
	return result
}

func (w *Window) msgBox(primarytext string, secondarytext string) (done chan struct{}) {
	done = make(chan struct{})
	go func() {
		<-_msgBox(w, primarytext, secondarytext, C.GtkMessageType(C.GTK_MESSAGE_OTHER), C.GtkButtonsType(C.GTK_BUTTONS_OK))
		done <- struct{}{}
	}()
	return done
}

func (w *Window) msgBoxError(primarytext string, secondarytext string) (done chan struct{}) {
	done = make(chan struct{})
	go func() {
		<-_msgBox(w, primarytext, secondarytext, C.GtkMessageType(C.GTK_MESSAGE_ERROR), C.GtkButtonsType(C.GTK_BUTTONS_OK))
		done <- struct{}{}
	}()
	return done
}
