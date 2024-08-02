
import os

from pyannote.audio import Pipeline
  
token = os.getenv("HF_TOKEHF_TOKEN")
pipeline = Pipeline.from_pretrained("pyannote/speaker-diarization-3.1", use_auth_token=token)

# inference on the whole file
pipeline("./files/1.wav")

# inference on an excerpt
from pyannote.core import Segment
excerpt = Segment(start=2.0, end=5.0)

from pyannote.audio import Audio
waveform, sample_rate = Audio().crop("./files/2.wav", excerpt)
pipeline({"waveform": waveform, "sample_rate": sample_rate})