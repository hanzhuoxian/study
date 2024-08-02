
    
from pyannote.audio import Model, Inference
import os

token = os.getenv("HF_TOKEHF_TOKENN")
model = Model.from_pretrained(
  "pyannote/segmentation-3.0",
  use_auth_token=token)
