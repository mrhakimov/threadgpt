import SwiftUI

struct ThreadGPTPalette {
    let background: Color
    let foreground: Color
    let card: Color
    let secondary: Color
    let muted: Color
    let mutedForeground: Color
    let border: Color
    let destructive: Color
    let destructiveForeground: Color
    let accent: Color
    let popover: Color

    static func resolve(_ scheme: ColorScheme) -> ThreadGPTPalette {
        switch scheme {
        case .dark:
            return ThreadGPTPalette(
                background: Color(hex: 0x0D1117),
                foreground: Color(hex: 0xD0D7DE),
                card: Color(hex: 0x161B22),
                secondary: Color(hex: 0x1C2333),
                muted: Color(hex: 0x1C2333),
                mutedForeground: Color(hex: 0x818B98),
                border: Color(hex: 0x30363D),
                destructive: Color(red: 0.73, green: 0.27, blue: 0.27),
                destructiveForeground: .white,
                accent: Color(hex: 0x1C2333),
                popover: Color(hex: 0x161B22)
            )
        default:
            return ThreadGPTPalette(
                background: .white,
                foreground: Color(hex: 0x0F0F0F),
                card: Color(hex: 0xF7F7F7),
                secondary: Color(hex: 0xEDEDED),
                muted: Color(hex: 0xEDEDED),
                mutedForeground: Color(hex: 0x616161),
                border: Color(hex: 0xE0E0E0),
                destructive: Color(red: 0.78, green: 0.38, blue: 0.38),
                destructiveForeground: .white,
                accent: Color(hex: 0xEDEDED),
                popover: .white
            )
        }
    }
}

extension Color {
    init(hex: UInt, alpha: Double = 1) {
        self.init(
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255,
            opacity: alpha
        )
    }
}

struct Hairline: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        ThreadGPTPalette.resolve(colorScheme).border.frame(height: 1)
    }
}

