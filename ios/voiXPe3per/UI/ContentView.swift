import SwiftUI

struct ContentView: View {
    @EnvironmentObject private var appState: AppState
    @State private var scannerVisible = false

    var body: some View {
        NavigationStack {
            VStack(spacing: 22) {
                VStack(spacing: 8) {
                    Text("voiXPe3per")
                        .font(.system(size: 34, weight: .bold, design: .rounded))
                    Text(appState.status)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                }

                if let trusted = appState.trustedDesktop {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Trusted desktop")
                            .font(.headline)
                        Text(trusted.room.isEmpty ? trusted.host : "Room \(trusted.room)")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                            .lineLimit(2)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding()
                    .background(Color(.secondarySystemBackground))
                    .clipShape(RoundedRectangle(cornerRadius: 12))

                    VStack(spacing: 8) {
                        Text("Mulai Screen Broadcast")
                            .font(.subheadline)
                            .fontWeight(.medium)
                        BroadcastPickerRepresentable()
                            .frame(width: 60, height: 60)
                    }
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color(.secondarySystemBackground))
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                }

                VStack(spacing: 12) {
                    Button {
                        scannerVisible = true
                    } label: {
                        Label("Scan QR Desktop", systemImage: "qrcode.viewfinder")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.large)
                    .disabled(appState.isPairing)

                    Button {
                        appState.reconnect()
                    } label: {
                        Label("Reconnect", systemImage: "bolt.horizontal")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.large)
                    .disabled(appState.trustedDesktop == nil || appState.isPairing)

                    Button(role: .destructive) {
                        appState.forgetDesktop()
                    } label: {
                        Label("Forget Desktop", systemImage: "trash")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .disabled(appState.trustedDesktop == nil || appState.isPairing)
                }

                Spacer()
            }
            .padding(24)
            .navigationTitle("Pairing")
            .sheet(isPresented: $scannerVisible) {
                QRScannerView { code in
                    scannerVisible = false
                    appState.pair(raw: code)
                }
            }
        }
    }
}

#Preview {
    ContentView()
        .environmentObject(AppState())
}
