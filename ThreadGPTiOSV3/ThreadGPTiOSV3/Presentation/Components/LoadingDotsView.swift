import SwiftUI

struct LoadingDotsView: View {
    @State private var dotCount = 0
    private let timer = Timer.publish(every: 0.4, on: .main, in: .common).autoconnect()

    var body: some View {
        HStack(spacing: 4) {
            ForEach(0..<3, id: \.self) { index in
                Circle()
                    .fill(Color.tgptMutedForeground)
                    .frame(width: 6, height: 6)
                    .opacity(dotCount % 3 == index ? 1.0 : 0.3)
            }
        }
        .onReceive(timer) { _ in
            dotCount += 1
        }
    }
}
