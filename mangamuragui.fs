// mangamuradlgui
// v1.0(180218) first created

namespace Firstform
open System
open System.Drawing
open System.Windows.Forms
open System.Diagnostics
open System.Text

module Main =

    let form = new Form(
        Text = "mangamuradl GUI v1.0(180218)"
    )

    let button = new Button(
        Text     = "Start",
        AutoSize = true,
        Anchor   = AnchorStyles.Left
    )

    let textbox = new TextBox(
        AutoSize = true,
        Dock = DockStyle.Fill
    )

    // log window
    let textbox2 = new TextBox(
        AutoSize   = true,
        Dock       = DockStyle.Fill,
        Multiline  = true,
        ScrollBars = ScrollBars.Vertical
    )

    // ____________________
    // | 1.start | 2.input |
    // |-------------------|
    // |  3.logwindow      |
    // |___________________|
    let tbl = new TableLayoutPanel(
        // CellBorderStyle = TableLayoutPanelCellBorderStyle.OutsetDouble,
        ColumnCount     = 2,
        RowCount        = 2,
        Dock            = DockStyle.Fill
    )

    form.Controls.Add(tbl)

    tbl.Controls.Add(button)
    tbl.Controls.Add(textbox)
    tbl.Controls.Add(textbox2)
    tbl.SetColumnSpan(textbox2, 2)


    let async1(syncContext, button : Button, inputbox : TextBox, logbox : TextBox) =
        async {
            button.Enabled <- false
            logbox.Text <- ""

            let startInfo = new ProcessStartInfo()
            startInfo.FileName               <- "mangamuradl"
            startInfo.Arguments              <- inputbox.Text
            startInfo.RedirectStandardOutput <- true
            startInfo.RedirectStandardError  <- true
            startInfo.StandardOutputEncoding <- Encoding.UTF8
            startInfo.StandardErrorEncoding  <- Encoding.UTF8
            startInfo.UseShellExecute        <- false
            startInfo.CreateNoWindow         <- true


            let handler = new DataReceivedEventHandler(fun sender e -> do
                // 何故か最後にハングする
                //logbox.AppendText(e.Data);logbox.AppendText(Environment.NewLine)
                // これだといける
                logbox.AppendText(e.Data + Environment.NewLine)
                ()
            )

            let proc = new Process(StartInfo = startInfo)

            proc.Start() |> ignore

            proc.OutputDataReceived.AddHandler(handler)
            proc.ErrorDataReceived.AddHandler(handler)

            proc.BeginOutputReadLine()
            proc.BeginErrorReadLine()

            proc.WaitForExit()

            proc.CancelOutputRead()
            proc.CancelErrorRead()

            button.Enabled <- true
        }

    let syncContext = System.Threading.SynchronizationContext()

    let buttonClick(sender:obj, args) =
        Async.Start(async1(syncContext, button, textbox, textbox2))
        ()

    button.Click.AddHandler(fun sender args -> buttonClick(sender, args))

    Application.Run(form)
