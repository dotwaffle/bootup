## 1. Candidate Matrix

- [x] 1.1 List representative chainload-shaped targets and the boot semantics each one requires.
- [x] 1.2 Select the first candidate route to test, including why it is the least invasive useful proof.
- [x] 1.3 Define the target-visible media or firmware state that must survive after Linux stage-1 exits.

## 2. Handoff Route Investigation

- [x] 2.1 Evaluate local u-root, Linux kexec, firmware, and bootloader facilities that could transfer control to the selected candidate.
- [x] 2.2 Record artifact provenance for any loader, firmware, ISO, disk, kernel, or RAM payload used by the investigation.
- [x] 2.3 Keep all generated or downloaded payloads outside tracked repository paths.

## 3. QEMU Proof

- [x] 3.1 Build a minimal QEMU command, helper script, or documented manual procedure for the selected route.
- [x] 3.2 Run the proof far enough to capture a target marker or the first hard blocker.
- [x] 3.3 Classify the result as viable, lab-only, or deferred based on target-environment evidence.

## 4. Recommendation

- [x] 4.1 Record the commands, observed output, and blocker or success marker in docs or the OpenSpec design.
- [x] 4.2 Recommend whether the next implementation should be a production chainload executor, another focused spike, or continued deferral.
- [x] 4.3 Confirm no generated bootloader binaries, distro payloads, ISOs, disk images, initramfs images, or firmware variable files are tracked.
