import SwiftUI

struct LoadingDotsView: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var animating = false

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        HStack(spacing: 5) {
            ForEach(0..<3, id: \.self) { index in
                Circle()
                    .fill(palette.mutedForeground.opacity(0.55))
                    .frame(width: 7, height: 7)
                    .offset(y: animating ? -4 : 0)
                    .animation(
                        .easeInOut(duration: 0.55)
                        .repeatForever(autoreverses: true)
                        .delay(Double(index) * 0.12),
                        value: animating
                    )
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .onAppear { animating = true }
    }
}

