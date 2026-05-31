import SwiftUI

@main
struct voiXPe3perApp: App {
    @StateObject private var appState = AppState()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(appState)
                .onOpenURL { url in
                    appState.pair(raw: url.absoluteString)
                }
        }
    }
}
