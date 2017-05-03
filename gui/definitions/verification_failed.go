package definitions

func init() {
	add(`VerificationFailed`, &defVerificationFailed{})
}

type defVerificationFailed struct{}

func (*defVerificationFailed) String() string {
	return `<interface>
  <object class="GtkDialog" id="dialog">
    <property name="window-position">GTK_WIN_POS_CENTER</property>
    <child internal-child="vbox">
      <object class="GtkBox" id="box">
        <property name="border-width">10</property>
        <property name="homogeneous">false</property>
        <property name="orientation">GTK_ORIENTATION_VERTICAL</property>
        <child>
          <object class="GtkGrid">
            <property name="halign">GTK_ALIGN_CENTER</property>
            <property name="row-spacing">60</property>
            <child>
              <object  class="GtkImage">
                <property name="file">build/images/red_alert.png</property>
              </object>
              <packing>
                <property name="left-attach">0</property>
                <property name="top-attach">0</property>
              </packing>
            </child>
            <child>
              <object class="GtkLabel">
                <property name="label" translatable="yes">Verification Failed</property>
              </object>
              <packing>
                <property name="left-attach">1</property>
                <property name="top-attach">0</property>
              </packing>
            </child>
          </object>
        </child>
        <child>
          <object class="GtkLabel" id="verification_message">
            <property name="label" translatable="yes"></property>
          </object>
        </child>
        <child>
          <object class="GtkGrid">
            <property name="column-spacing">6</property>
            <property name="row-spacing">2</property>
            <child>
              <object class="GtkLabel">
                <property name="visible">True</property>
                <property name="label" translatable="yes"> - The peer that sent the PIN sent the incorrect PIN</property>
                <property name="justify">GTK_JUSTIFY_LEFT</property>
                <property name="halign">GTK_ALIGN_START</property>
              </object>
              <packing>
                <property name="left-attach">0</property>
                <property name="top-attach">0</property>
              </packing>
            </child>
            <child>
              <object class="GtkLabel">
                <property name="visible">True</property>
                <property name="label" translatable="yes"> - The peer that entered the PIN did so incorrectly</property>
                <property name="justify">GTK_JUSTIFY_LEFT</property>
                <property name="halign">GTK_ALIGN_START</property>
              </object>
              <packing>
                <property name="left-attach">0</property>
                <property name="top-attach">1</property>
              </packing>
            </child>
            <child>
              <object class="GtkLabel">
                <property name="visible">True</property>
                <property name="label" translatable="yes"> - Someone is listening in on this conversation</property>
                <property name="justify">GTK_JUSTIFY_LEFT</property>
                <property name="halign">GTK_ALIGN_START</property>
              </object>
              <packing>
                <property name="left-attach">0</property>
                <property name="top-attach">2</property>
              </packing>
            </child>
            <child internal-child="action_area">
              <object class="GtkButtonBox" id="button_box">
                <property name="orientation">GTK_ORIENTATION_HORIZONTAL</property>
                <child>
                  <object class="GtkButton" id="try_later">
                    <property name="can-default">true</property>
                    <property name="label" translatable="yes">Do this later</property>
                  </object>
                </child>
              </object>
            </child>
          </object>
        </child>
      </object>
    </child>
  </object>
</interface>`
}
