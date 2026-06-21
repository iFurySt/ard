package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ifuryst/ard/internal/catalog"
	"github.com/ifuryst/ard/internal/config"
	"github.com/ifuryst/ard/internal/policy"
	"github.com/ifuryst/ard/internal/verify"
	"github.com/spf13/cobra"
)

func newVerifyCommand(root *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "verify",
		Short: "Verify ARD resources",
	}
	command.AddCommand(newVerifyCatalogCommand(root))
	return command
}

func newVerifyCatalogCommand(root *rootOptions) *cobra.Command {
	var jsonOutput bool
	var verifySourceDigests bool
	var requireSourceDigests bool
	var verifyAttestationDigests bool
	var requireAttestationDigests bool
	var verifyProvenanceDigests bool
	var requireProvenanceDigests bool
	var jwsTrustAnchors string
	var jwsRemoteJWKS []string
	var jwsDiscoverDIDWeb bool
	var jwsDiscoverOIDC bool
	var jwsDiscoverTLSCert bool
	var requireJWSSignatures bool
	command := &cobra.Command{
		Use:   "catalog SOURCE",
		Short: "Verify an ai-catalog.json file or URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			loadedCatalog, err := catalog.Load(ctx, args[0])
			if err != nil {
				return err
			}
			sourceDigestResults := []verify.SourceDigestResult{}
			if verifySourceDigests || requireSourceDigests {
				results, err := verify.VerifySourceDigestsWithOptions(ctx, loadedCatalog, verify.SourceDigestOptions{
					RequirePinnedURLArtifacts: requireSourceDigests,
				})
				if err != nil {
					return err
				}
				sourceDigestResults = results
			}
			attestationDigestResults := []verify.AttestationDigestResult{}
			if verifyAttestationDigests || requireAttestationDigests {
				results, err := verify.VerifyAttestationDigestsWithOptions(ctx, loadedCatalog, verify.AttestationDigestOptions{
					RequirePinnedAttestations: requireAttestationDigests,
				})
				if err != nil {
					return err
				}
				attestationDigestResults = results
			}
			provenanceDigestResults := []verify.ProvenanceDigestResult{}
			if verifyProvenanceDigests || requireProvenanceDigests {
				results, err := verify.VerifyProvenanceDigestsWithOptions(ctx, loadedCatalog, verify.ProvenanceDigestOptions{
					RequirePinnedHTTPSourceIDs: requireProvenanceDigests,
				})
				if err != nil {
					return err
				}
				provenanceDigestResults = results
			}
			signatureResults := []verify.SignatureResult{}
			if jwsTrustAnchors != "" || len(jwsRemoteJWKS) > 0 || jwsDiscoverDIDWeb || jwsDiscoverOIDC || jwsDiscoverTLSCert || requireJWSSignatures {
				anchorSets := []verify.TrustAnchors{}
				if jwsTrustAnchors != "" {
					anchors, err := verify.LoadTrustAnchors(jwsTrustAnchors)
					if err != nil {
						return err
					}
					anchorSets = append(anchorSets, anchors)
				}
				for _, remoteJWKS := range jwsRemoteJWKS {
					anchors, err := verify.LoadRemoteTrustAnchors(ctx, remoteJWKS, nil)
					if err != nil {
						return fmt.Errorf("load remote JWKS %s: %w", remoteJWKS, err)
					}
					anchorSets = append(anchorSets, anchors)
				}
				if jwsDiscoverDIDWeb {
					anchors, err := verify.DiscoverDIDWebTrustAnchors(ctx, loadedCatalog, nil)
					if err != nil {
						return fmt.Errorf("discover did:web trust anchors: %w", err)
					}
					anchorSets = append(anchorSets, anchors)
				}
				if jwsDiscoverOIDC {
					anchors, err := verify.DiscoverOIDCTrustAnchors(ctx, loadedCatalog, nil)
					if err != nil {
						return fmt.Errorf("discover OIDC trust anchors: %w", err)
					}
					anchorSets = append(anchorSets, anchors)
				}
				if jwsDiscoverTLSCert {
					anchors, err := verify.DiscoverTLSCertificateTrustAnchors(ctx, loadedCatalog, nil)
					if err != nil {
						return fmt.Errorf("discover TLS certificate trust anchors: %w", err)
					}
					anchorSets = append(anchorSets, anchors)
				}
				anchors := verify.MergeTrustAnchors(anchorSets...)
				if len(anchors.Keys) == 0 {
					return fmt.Errorf("--jws-trust-anchors, --jws-remote-jwks, --jws-discover-did-web, --jws-discover-oidc, or --jws-discover-tls-cert is required when verifying JWS signatures")
				}
				results, err := verify.VerifySignatures(loadedCatalog, verify.SignatureOptions{
					RequireSignatures: requireJWSSignatures,
					TrustAnchors:      anchors,
				})
				if err != nil {
					return err
				}
				signatureResults = results
			}
			policyEvaluations := []policy.Evaluation{}
			policyFile := config.PolicyFile(root.policyFile)
			if policyFile != "" {
				loadedPolicy, err := policy.LoadFile(policyFile)
				if err != nil {
					return fmt.Errorf("load policy: %w", err)
				}
				_, evaluations, err := loadedPolicy.EvaluateCatalog(loadedCatalog)
				if err != nil {
					return err
				}
				policyEvaluations = evaluations
			}
			if jsonOutput {
				payload := map[string]any{
					"valid":                      true,
					"specVersion":                loadedCatalog.SpecVersion,
					"entries":                    len(loadedCatalog.Entries),
					"sourceDigestsVerified":      len(sourceDigestResults),
					"attestationDigestsVerified": len(attestationDigestResults),
					"provenanceDigestsVerified":  len(provenanceDigestResults),
					"signaturesVerified":         len(signatureResults),
				}
				if policyFile != "" {
					payload["policyEvaluated"] = true
					payload["policyEntries"] = len(policyEvaluations)
					payload["policy"] = policyEvaluations
				}
				if verifySourceDigests {
					payload["sourceDigests"] = sourceDigestResults
				}
				if requireSourceDigests {
					payload["sourceDigestsRequired"] = true
					payload["sourceDigests"] = sourceDigestResults
				}
				if verifyAttestationDigests {
					payload["attestationDigests"] = attestationDigestResults
				}
				if requireAttestationDigests {
					payload["attestationDigestsRequired"] = true
					payload["attestationDigests"] = attestationDigestResults
				}
				if verifyProvenanceDigests {
					payload["provenanceDigests"] = provenanceDigestResults
				}
				if requireProvenanceDigests {
					payload["provenanceDigestsRequired"] = true
					payload["provenanceDigests"] = provenanceDigestResults
				}
				if jwsTrustAnchors != "" || len(jwsRemoteJWKS) > 0 || jwsDiscoverDIDWeb || jwsDiscoverOIDC || jwsDiscoverTLSCert {
					payload["signatures"] = signatureResults
				}
				if requireJWSSignatures {
					payload["signaturesRequired"] = true
					payload["signatures"] = signatureResults
				}
				encoded, err := json.MarshalIndent(payload, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
				return nil
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"valid ai-catalog.json: %d entries\n",
				len(loadedCatalog.Entries),
			)
			if verifySourceDigests || requireSourceDigests {
				fmt.Fprintf(cmd.OutOrStdout(), "verified source digests: %d\n", len(sourceDigestResults))
			}
			if requireSourceDigests {
				fmt.Fprintf(cmd.OutOrStdout(), "required source digests: true\n")
			}
			if verifyAttestationDigests || requireAttestationDigests {
				fmt.Fprintf(cmd.OutOrStdout(), "verified attestation digests: %d\n", len(attestationDigestResults))
			}
			if requireAttestationDigests {
				fmt.Fprintf(cmd.OutOrStdout(), "required attestation digests: true\n")
			}
			if verifyProvenanceDigests || requireProvenanceDigests {
				fmt.Fprintf(cmd.OutOrStdout(), "verified provenance digests: %d\n", len(provenanceDigestResults))
			}
			if requireProvenanceDigests {
				fmt.Fprintf(cmd.OutOrStdout(), "required provenance digests: true\n")
			}
			if jwsTrustAnchors != "" || len(jwsRemoteJWKS) > 0 || jwsDiscoverDIDWeb || jwsDiscoverOIDC || jwsDiscoverTLSCert || requireJWSSignatures {
				fmt.Fprintf(cmd.OutOrStdout(), "verified signatures: %d\n", len(signatureResults))
			}
			if requireJWSSignatures {
				fmt.Fprintf(cmd.OutOrStdout(), "required signatures: true\n")
			}
			if policyFile != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "policy entries evaluated: %d\n", len(policyEvaluations))
			}
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print machine-readable verification result")
	command.Flags().BoolVar(&verifySourceDigests, "source-digests", false, "Fetch URL artifacts and verify trustManifest.sourceDigest")
	command.Flags().BoolVar(&requireSourceDigests, "require-source-digests", false, "Require every URL artifact to have trustManifest.sourceDigest and verify it")
	command.Flags().BoolVar(&verifyAttestationDigests, "attestation-digests", false, "Fetch attestation documents and verify trustManifest.attestations[].digest")
	command.Flags().BoolVar(&requireAttestationDigests, "require-attestation-digests", false, "Require every trustManifest attestation to have digest and verify it")
	command.Flags().BoolVar(&verifyProvenanceDigests, "provenance-digests", false, "Fetch HTTP(S) provenance sourceIds and verify trustManifest.provenance[].sourceDigest")
	command.Flags().BoolVar(&requireProvenanceDigests, "require-provenance-digests", false, "Require every HTTP(S) trustManifest provenance sourceId to have sourceDigest and verify it")
	command.Flags().StringVar(&jwsTrustAnchors, "jws-trust-anchors", "", "JSON trust anchors for verifying detached JWS trustManifest.signature values")
	command.Flags().StringArrayVar(&jwsRemoteJWKS, "jws-remote-jwks", nil, "HTTPS JWKS URL for verifying detached JWS trustManifest.signature values")
	command.Flags().BoolVar(&jwsDiscoverDIDWeb, "jws-discover-did-web", false, "Discover did:web DID document keys for verifying detached JWS trustManifest.signature values")
	command.Flags().BoolVar(&jwsDiscoverOIDC, "jws-discover-oidc", false, "Discover OpenID Connect jwks_uri keys for verifying detached JWS trustManifest.signature values")
	command.Flags().BoolVar(&jwsDiscoverTLSCert, "jws-discover-tls-cert", false, "Discover HTTPS TLS leaf certificate keys for verifying detached JWS trustManifest.signature values")
	command.Flags().BoolVar(&requireJWSSignatures, "require-jws-signatures", false, "Require every catalog entry to have a verifiable detached JWS trustManifest.signature")
	return command
}
