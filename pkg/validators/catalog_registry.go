//go:build !osspublic

package validators

// AllCore returns all Core validators (Iteration 1–3 + Phase A + Phase B).
func AllCore() []Validator {
	iter2 := AllIter2()
	iter3 := AllIter3()
	phaseA := AllPhaseA()
	phaseB := AllPhaseB()
	out := make([]Validator, 0, len(iter2)+len(iter3)+len(phaseA)+len(phaseB))
	out = append(out, iter2...)
	out = append(out, iter3...)
	out = append(out, phaseA...)
	out = append(out, phaseB...)
	out = append(out, AllStarterV1Core()...)
	return out
}

// AllIter2 returns the 25 Core validators from Iteration 2.
func AllIter2() []Validator {
	return []Validator{
		ValidFormat{},
		RequiredFields{},
		TypeChecking{},
		EnumValidator{},
		ListLength{},
		FieldPresenceRules{},
		MaxLength{},
		MinLength{},
		RequiredSections{},
		MarkdownFormat{},
		RegexPattern{},
		ForbiddenPatterns{},
		RangeValidator{},
		NumericConsistency{},
		NumericRelations{},
		CrossFieldLogic{},
		DependencyRules{},
		PIIDetection{},
		SQLInjection{},
		PromptInjection{},
		URLSafety{},
		NumericMatch{},
		KeywordPreservation{},
		LatencyValidator{},
		TokenCountValidator{},
	}
}

// AllIter3 returns the 32 Core validators added in Iteration 3.
func AllIter3() []Validator {
	return []Validator{
		NoRefusalPhrases{},
		NoMetaCommentary{},
		NoSelfReference{},
		NoApologies{},
		NoHedging{},
		NoThinkingTagLeakage{},
		NoSystemPromptLeakage{},
		NoRoleMarkerLeakage{},
		NoTruncationMarkers{},
		NoPlaceholderText{},
		NoLoremIpsum{},
		CompleteSentenceEnding{},
		NoRepeatedPhrases{},
		RepetitionRatioMax{},
		MarkdownLinkWellFormed{},
		CodeBlockBalanced{},
		BracketBalanced{},
		ValidUUID{},
		ValidEmailRFC5322{},
		ValidURLRFC3986{},
		ValidPhoneE164{},
		ValidISODate{},
		DecimalPrecisionMax{},
		IntegerOnly{},
		NonNegative{},
		CurrencyFormat{},
		PercentageFormat{},
		SecretsDetection{},
		XSSPattern{},
		PathTraversal{},
		CostPerRequestMaxUSD{},
		ModelWhitelistPerAgent{},
	}
}

// AllPhaseA returns deterministic validators shipped in Phase A (no AI-as-judge).
func AllPhaseA() []Validator {
	return []Validator{
		JSONNoExtraFields{},
		ValidBase64{},
		ValidJWTStructure{},
		ValidSemver{},
		ValidColorHex{},
		ValidIPv4OrIPv6{},
		MarkdownImageWellFormed{},
		MarkdownTableColumnsConsistent{},
		CodeBlockLanguageWhitelist{},
		InstructionCompliance{},
		ForbiddenContent{},
		SensitiveContent{},
		ToneStyleRules{},
		OutputShape{},
		NoDuplicateJSONKeys{},
		WordCountRange{},
		NoMarkdownFencedWrapper{},
	}
}

// AllPhaseB returns deterministic validators shipped in Phase B (RAG, input fidelity, schema).
func AllPhaseB() []Validator {
	return []Validator{
		FactMatch{},
		InputCoverage{},
		InputOutputDiff{},
		StepConsistency{},
		RequiredStepOrder{},
		StateTransitionRules{},
		MultiStepNumericLogic{},
		RAGSourcePresence{},
		RAGSourceCoverage{},
		RAGSourceMatch{},
		CitationFormatValid{},
		RetryCountValidator{},
		HomoglyphDetection{},
		JSONSchemaValidate{},
		HTMLNoScript{},
		ValidVATEU{},
		AIDisclosureRequired{},
		BranchOutputsPresent{},
		CrossBranchFieldMatch{},
		BranchPassCount{},
	}
}

// AllCompliance returns Compliance & Industry validators (Iteration 4a + geo).
func AllCompliance() []Validator {
	base := []Validator{
		PIIPesel{},
		PIINIP{},
		PIISsnUS{},
		PIIEINUS{},
		PIIITINUS{},
		PIIMedicareMBI{},
		PIINHSNumber{},
		PIIAHV{},
		PHIUSHipaaPatterns{},
		PCIDSSPatterns{},
		ValidIBAN{},
		ValidCreditCardLuhn{},
		ValidISBN{},
		ValidPESEL{},
		ValidNIP{},
		ValidREGON{},
		ValidPostalCode{},
		ValidISOCurrency{},
		ValidISOCountry{},
		ValidISOLanguage{},
		DateInRange{},
		DateFutureOnly{},
		DatePastOnly{},
		BusinessDayOnly{},
		TimezonePresent{},
		AgeFromBirthdateInRange{},
		MedicalDisclaimerRequired{},
		FinancialDisclaimerRequired{},
		UnsubscribeLinkRequired{},
		CommandInjectionPattern{},
		TemplateInjection{},
		InternalNetworkURL{},
		XXEPattern{},
		GeoCountryWhitelist{},
		GeoCoordinatesInBounds{},
		GeoForbiddenCountries{},
	}
	out := make([]Validator, 0, len(base)+3)
	out = append(out, base...)
	out = append(out, AllStarterV1Compliance()...)
	return out
}

// AllRegistered returns Core + Compliance + Code validators (full catalog).
func AllRegistered() []Validator {
	out := make([]Validator, 0, len(AllCore())+len(AllCompliance())+len(AllCode()))
	out = append(out, AllCore()...)
	out = append(out, AllCompliance()...)
	out = append(out, AllCode()...)
	return out
}

// NewCatalogRegistry builds the full production validator registry.
func NewCatalogRegistry() *Registry {
	return NewRegistry(AllRegistered()...)
}
