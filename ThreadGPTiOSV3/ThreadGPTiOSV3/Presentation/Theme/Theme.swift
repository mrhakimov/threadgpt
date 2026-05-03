import SwiftUI

enum Theme {
    // MARK: - Light Mode Colors (matching web frontend globals.css)

    // Background: #FFFFFF / Dark: #0D1117
    static let background = Color("background")
    // Foreground: #0F0F0F / Dark: #D0D7DE
    static let foreground = Color("foreground")
    // Card: #F7F7F7 / Dark: #161B22
    static let card = Color("card")
    // Secondary: #EDEDED / Dark: #1C2333
    static let secondary = Color("secondary")
    // Muted foreground: #616161 / Dark: #818B98
    static let mutedForeground = Color("mutedForeground")
    // Border: #E0E0E0 / Dark: #30363D
    static let border = Color("border")
    // Destructive
    static let destructive = Color("destructive")

    // Convenience: user bubble uses secondary, assistant bubble uses card
    static let userBubble = secondary
    static let assistantBubble = card
}

// MARK: - Programmatic colors for when asset catalog colors aren't loaded

extension Color {
    static let tgptBackground = Color(light: .white, dark: Color(hex: 0x0D1117))
    static let tgptForeground = Color(light: Color(hex: 0x0F0F0F), dark: Color(hex: 0xD0D7DE))
    static let tgptCard = Color(light: Color(hex: 0xF7F7F7), dark: Color(hex: 0x161B22))
    static let tgptSecondary = Color(light: Color(hex: 0xEDEDED), dark: Color(hex: 0x1C2333))
    static let tgptMutedForeground = Color(light: Color(hex: 0x616161), dark: Color(hex: 0x818B98))
    static let tgptBorder = Color(light: Color(hex: 0xE0E0E0), dark: Color(hex: 0x30363D))
    static let tgptDestructive = Color(light: Color(hue: 0, saturation: 0.55, brightness: 0.58),
                                        dark: Color(hue: 0, saturation: 0.45, brightness: 0.50))
}

extension Color {
    init(hex: UInt, alpha: Double = 1.0) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255,
            opacity: alpha
        )
    }

    init(light: Color, dark: Color) {
        self.init(uiColor: UIColor { traits in
            traits.userInterfaceStyle == .dark ? UIColor(dark) : UIColor(light)
        })
    }
}
