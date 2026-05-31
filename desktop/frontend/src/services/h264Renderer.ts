import type { StreamFrame } from "../types";

type VideoDecoderLike = {
  configure(config: { codec: string; optimizeForLatency?: boolean }): void;
  decode(chunk: unknown): void;
  close(): void;
};

export class H264Renderer {
  private canvas: HTMLCanvasElement | null = null;
  private context: CanvasRenderingContext2D | null = null;
  private decoder: VideoDecoderLike | null = null;
  private fallbackTick = 0;

  attach(canvas: HTMLCanvasElement | null) {
    this.canvas = canvas;
    this.context = canvas?.getContext("2d") ?? null;
    if (!canvas) {
      this.decoder?.close();
      this.decoder = null;
      return;
    }

    this.drawIdle();
    this.prepareDecoder();
  }

  push(frame: StreamFrame) {
    if (!this.canvas || !this.context) {
      return;
    }

    const bytes = base64ToBytes(frame.data);
    if (this.decoder) {
      try {
        const EncodedVideoChunkCtor = webCodecs().EncodedVideoChunk;
        if (!EncodedVideoChunkCtor) {
          throw new Error("EncodedVideoChunk unavailable");
        }
        this.decoder.decode(
          new EncodedVideoChunkCtor({
            type: frame.keyFrame ? "key" : "delta",
            timestamp: Math.floor(frame.timestampNs / 1000),
            data: bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
          })
        );
        return;
      } catch {
        this.decoder.close();
        this.decoder = null;
      }
    }

    this.drawFallback(frame);
  }

  private prepareDecoder() {
    const VideoDecoderCtor = webCodecs().VideoDecoder;
    if (!VideoDecoderCtor || !this.canvas || !this.context) {
      return;
    }

    this.decoder = new VideoDecoderCtor({
      output: (videoFrame: {
        close(): void;
      }) => {
        if (!this.canvas || !this.context) {
          videoFrame.close();
          return;
        }
        this.context.drawImage(videoFrame as unknown as CanvasImageSource, 0, 0, this.canvas.width, this.canvas.height);
        videoFrame.close();
      },
      error: () => {
        this.decoder?.close();
        this.decoder = null;
      }
    });
    this.decoder.configure({ codec: "avc1.42E01E", optimizeForLatency: true });
  }

  private drawIdle() {
    if (!this.canvas || !this.context) {
      return;
    }
    const { width, height } = this.canvas;
    this.context.fillStyle = "#0b0c0f";
    this.context.fillRect(0, 0, width, height);
    this.context.strokeStyle = "#242a31";
    this.context.lineWidth = 1;
    for (let x = 0; x < width; x += 32) {
      this.context.beginPath();
      this.context.moveTo(x, 0);
      this.context.lineTo(x, height);
      this.context.stroke();
    }
  }

  private drawFallback(frame: StreamFrame) {
    if (!this.canvas || !this.context) {
      return;
    }
    const { width, height } = this.canvas;
    this.fallbackTick = (this.fallbackTick + 1) % width;

    this.context.fillStyle = "#0b0c0f";
    this.context.fillRect(0, 0, width, height);
    this.context.fillStyle = "#151922";
    this.context.fillRect(0, 0, width, height);
    this.context.fillStyle = frame.keyFrame ? "#4fd18b" : "#66d9ef";
    this.context.fillRect(this.fallbackTick, 0, 4, height);
    this.context.fillStyle = "#d6dde8";
    this.context.font = "14px Consolas, monospace";
    this.context.fillText(`H264 ${frame.keyFrame ? "key" : "delta"} frame received`, 24, 32);
  }
}

function base64ToBytes(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}

function webCodecs(): {
  VideoDecoder?: new (init: {
    output(frame: { close(): void }): void;
    error(error: Error): void;
  }) => VideoDecoderLike;
  EncodedVideoChunk?: new (init: {
    type: "key" | "delta";
    timestamp: number;
    data: BufferSource;
  }) => unknown;
} {
  return window as unknown as {
    VideoDecoder?: new (init: {
      output(frame: { close(): void }): void;
      error(error: Error): void;
    }) => VideoDecoderLike;
    EncodedVideoChunk?: new (init: {
      type: "key" | "delta";
      timestamp: number;
      data: BufferSource;
    }) => unknown;
  };
}
