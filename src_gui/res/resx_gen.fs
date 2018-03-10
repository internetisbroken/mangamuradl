// 180308 created

module resx_gen

open System
open System.IO
open System.Drawing
open System.Resources
open System.Text.RegularExpressions

[<EntryPoint>]
let main args =
    let version =
        use sr = new StreamReader(args.[0])
        let content = sr.ReadToEnd()
        let ma = Regex.Match(content, @"var\s+VERSION\s*=\s*""([^""]*)""")
        ma.Groups.[1].Value

    if String.Compare(version, "") <> 0 then
        let typeName =
            String.Format("{0}, {1}",
                typeof<Icon>.FullName,
                typeof<Icon>.Assembly.FullName
            )
        let icon = new ResXFileRef("icon0.ico", typeName)
        let resx = new ResXResourceWriter("mangamuradl-gui.resx")
        resx.AddResource("resCliVersion", version);
        resx.AddResource("resIcon0", icon)
        resx.Close()

        printfn "VERSION: %s" version
        0
    else
        printfn "VERSION not found"
        1
